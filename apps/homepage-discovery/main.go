package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	annotationEnabled     = "gethomepage.dev/enabled"
	annotationName        = "gethomepage.dev/name"
	annotationDescription = "gethomepage.dev/description"
	annotationGroup       = "gethomepage.dev/group"
	annotationIcon        = "gethomepage.dev/icon"
	annotationHref        = "gethomepage.dev/href"
	annotationWeight      = "gethomepage.dev/weight"

	defaultGroup = "Discovered"
)

type ServiceEntry struct {
	Description string `yaml:"description,omitempty"`
	Href        string `yaml:"href"`
	Icon        string `yaml:"icon,omitempty"`
}

type DiscoveredService struct {
	Name        string
	Description string
	Group       string
	Icon        string
	Href        string
	Weight      int
}

type Controller struct {
	clientset         *kubernetes.Clientset
	homepageNamespace string
	homepageConfigMap string
	restartHomepage   bool
	discoveredGroups  map[string]bool // Groups managed by this controller
}

func main() {
	fmt.Println("Homepage Discovery Controller starting...")

	// Get configuration from environment
	homepageNamespace := getEnv("HOMEPAGE_NAMESPACE", "homepage")
	homepageConfigMap := getEnv("HOMEPAGE_CONFIGMAP", "homepage")
	restartHomepage := getEnv("RESTART_HOMEPAGE", "true") == "true"

	// Create Kubernetes client
	clientset, err := createClient()
	if err != nil {
		fmt.Printf("Error creating Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	controller := &Controller{
		clientset:         clientset,
		homepageNamespace: homepageNamespace,
		homepageConfigMap: homepageConfigMap,
		restartHomepage:   restartHomepage,
		discoveredGroups:  make(map[string]bool),
	}

	// Create informer factory
	factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)
	serviceInformer := factory.Core().V1().Services().Informer()

	// Add event handlers
	serviceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.handleServiceChange()
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			controller.handleServiceChange()
		},
		DeleteFunc: func(obj interface{}) {
			controller.handleServiceChange()
		},
	})

	// Start informer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go factory.Start(ctx.Done())

	// Wait for cache sync
	if !cache.WaitForCacheSync(ctx.Done(), serviceInformer.HasSynced) {
		fmt.Println("Error waiting for cache sync")
		os.Exit(1)
	}

	fmt.Println("Cache synced, controller running...")

	// Initial sync
	controller.handleServiceChange()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("Shutting down...")
}

func createClient() (*kubernetes.Clientset, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}
	return kubernetes.NewForConfig(config)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Controller) handleServiceChange() {
	ctx := context.Background()

	// List all services
	services, err := c.clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing services: %v\n", err)
		return
	}

	// Collect discovered services
	discovered := make(map[string][]DiscoveredService)
	for _, svc := range services.Items {
		if svc.Annotations[annotationEnabled] != "true" {
			continue
		}

		ds := DiscoveredService{
			Name:        svc.Annotations[annotationName],
			Description: svc.Annotations[annotationDescription],
			Group:       svc.Annotations[annotationGroup],
			Icon:        svc.Annotations[annotationIcon],
			Href:        svc.Annotations[annotationHref],
		}

		// Use service name if no name annotation
		if ds.Name == "" {
			ds.Name = svc.Name
		}

		// Use default group if not specified
		if ds.Group == "" {
			ds.Group = defaultGroup
		}

		// Skip if no href
		if ds.Href == "" {
			fmt.Printf("Skipping service %s/%s: no href annotation\n", svc.Namespace, svc.Name)
			continue
		}

		c.discoveredGroups[ds.Group] = true
		discovered[ds.Group] = append(discovered[ds.Group], ds)

		fmt.Printf("Discovered service: %s (%s) -> %s\n", ds.Name, ds.Group, ds.Href)
	}

	// Sort services within each group by name
	for group := range discovered {
		sort.Slice(discovered[group], func(i, j int) bool {
			return discovered[group][i].Name < discovered[group][j].Name
		})
	}

	// Update ConfigMap
	if err := c.updateConfigMap(ctx, discovered); err != nil {
		fmt.Printf("Error updating ConfigMap: %v\n", err)
		return
	}

	// Restart homepage if configured
	if c.restartHomepage {
		if err := c.restartHomepagePod(ctx); err != nil {
			fmt.Printf("Error restarting homepage: %v\n", err)
		}
	}
}

func (c *Controller) updateConfigMap(ctx context.Context, discovered map[string][]DiscoveredService) error {
	// Get current ConfigMap
	cm, err := c.clientset.CoreV1().ConfigMaps(c.homepageNamespace).Get(ctx, c.homepageConfigMap, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	// Parse current services.yaml
	servicesYaml := cm.Data["services.yaml"]
	var services []map[string][]map[string]ServiceEntry
	if servicesYaml != "" {
		if err := yaml.Unmarshal([]byte(servicesYaml), &services); err != nil {
			return fmt.Errorf("failed to parse services.yaml: %w", err)
		}
	}

	// Remove previously discovered groups (groups managed by this controller)
	var filteredServices []map[string][]map[string]ServiceEntry
	for _, groupMap := range services {
		for groupName := range groupMap {
			if !c.discoveredGroups[groupName] {
				filteredServices = append(filteredServices, groupMap)
			}
		}
	}

	// Add discovered services
	for groupName, groupServices := range discovered {
		groupEntries := make([]map[string]ServiceEntry, 0, len(groupServices))
		for _, svc := range groupServices {
			entry := map[string]ServiceEntry{
				svc.Name: {
					Description: svc.Description,
					Href:        svc.Href,
					Icon:        svc.Icon,
				},
			}
			groupEntries = append(groupEntries, entry)
		}
		if len(groupEntries) > 0 {
			filteredServices = append(filteredServices, map[string][]map[string]ServiceEntry{
				groupName: groupEntries,
			})
		}
	}

	// Marshal back to YAML
	newServicesYaml, err := yaml.Marshal(filteredServices)
	if err != nil {
		return fmt.Errorf("failed to marshal services.yaml: %w", err)
	}

	// Check if changed
	if string(newServicesYaml) == servicesYaml {
		fmt.Println("No changes to services.yaml")
		return nil
	}

	// Update ConfigMap
	cm.Data["services.yaml"] = string(newServicesYaml)

	// Also update settings.yaml to include discovered groups in layout
	if err := c.updateSettingsLayout(cm, discovered); err != nil {
		fmt.Printf("Warning: failed to update settings layout: %v\n", err)
	}

	_, err = c.clientset.CoreV1().ConfigMaps(c.homepageNamespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}

	fmt.Printf("Updated ConfigMap with %d groups\n", len(discovered))
	return nil
}

func (c *Controller) updateSettingsLayout(cm *corev1.ConfigMap, discovered map[string][]DiscoveredService) error {
	settingsYaml := cm.Data["settings.yaml"]
	if settingsYaml == "" {
		return nil
	}

	var settings map[string]interface{}
	if err := yaml.Unmarshal([]byte(settingsYaml), &settings); err != nil {
		return err
	}

	layout, ok := settings["layout"].([]interface{})
	if !ok {
		return nil
	}

	// Check which discovered groups are already in layout
	existingGroups := make(map[string]bool)
	for _, item := range layout {
		if groupMap, ok := item.(map[string]interface{}); ok {
			for groupName := range groupMap {
				existingGroups[groupName] = true
			}
		}
	}

	// Add missing discovered groups to layout
	modified := false
	for groupName := range discovered {
		if !existingGroups[groupName] {
			layout = append(layout, map[string]interface{}{
				groupName: map[string]interface{}{
					"style":   "row",
					"columns": 4,
				},
			})
			modified = true
		}
	}

	if modified {
		settings["layout"] = layout
		newSettingsYaml, err := yaml.Marshal(settings)
		if err != nil {
			return err
		}
		cm.Data["settings.yaml"] = string(newSettingsYaml)
	}

	return nil
}

func (c *Controller) restartHomepagePod(ctx context.Context) error {
	// Patch deployment to trigger rollout
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"homepage-discovery/restartedAt": time.Now().Format(time.RFC3339),
					},
				},
			},
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	_, err = c.clientset.AppsV1().Deployments(c.homepageNamespace).Patch(
		ctx,
		"homepage",
		types.StrategicMergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)

	if err != nil {
		return err
	}

	fmt.Println("Triggered homepage restart")
	return nil
}
