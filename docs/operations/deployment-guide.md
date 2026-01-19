# Deployment Guide

This guide provides step-by-step instructions for bootstrapping the Kubernetes cluster from scratch, including all the manual steps and workarounds discovered during deployment.

## Prerequisites

- Kubernetes cluster running (k3s, kind, EKS, GKE, etc.)
- `kubectl` configured with cluster access
- Cloudflare account with a Tunnel token from [Zero Trust Dashboard](https://one.dash.cloudflare.com/)

## Bootstrap Procedure

### Step 1: Create Cloudflare Namespace and Secret

The cloudflared deployment requires a secret containing the tunnel token. Create this first:

```bash
# Create the namespace
kubectl create namespace cloudflare

# Create the secret with your tunnel token
kubectl create secret generic cloudflare-tunnel-token \
  --from-literal=token=<your-tunnel-token> \
  -n cloudflare
```

!!! warning "Token Security"
    Never commit your tunnel token to Git. The secret must be created manually or via a secrets management solution.

### Step 2: Apply ArgoCD from Upstream

Deploy ArgoCD using the bootstrap kustomization:

```bash
kubectl apply -k argocd/bootstrap/
```

This applies:

- ArgoCD v2.9.3 from upstream manifests
- Namespace configuration
- App-of-apps Application
- IngressRoute for Traefik

### Step 3: Apply ConfigMap Patches with Correct Labels

The upstream ArgoCD manifests include ConfigMaps that need to be patched. However, patches sometimes fail if labels are missing. If you see ConfigMap errors, apply the patches manually:

```bash
# Check if ConfigMaps exist
kubectl get configmap -n argocd

# If argocd-cm exists but patches failed, apply with labels
kubectl patch configmap argocd-cm -n argocd \
  --type merge \
  -p '{"metadata":{"labels":{"app.kubernetes.io/part-of":"argocd"}}}'

kubectl patch configmap argocd-rbac-cm -n argocd \
  --type merge \
  -p '{"metadata":{"labels":{"app.kubernetes.io/part-of":"argocd"}}}'

# Now apply custom configuration
kubectl apply -f argocd/bootstrap/patches/argocd-cm.yaml
kubectl apply -f argocd/bootstrap/patches/argocd-rbac-cm.yaml
```

### Step 4: Delete Network Policies That Block Access

ArgoCD installs network policies that may block ingress traffic. If you cannot access ArgoCD, check and delete restrictive policies:

```bash
# List network policies in argocd namespace
kubectl get networkpolicy -n argocd

# Delete the default ArgoCD network policies if they block traffic
kubectl delete networkpolicy argocd-application-controller-network-policy -n argocd
kubectl delete networkpolicy argocd-applicationset-controller-network-policy -n argocd
kubectl delete networkpolicy argocd-dex-server-network-policy -n argocd
kubectl delete networkpolicy argocd-notifications-controller-network-policy -n argocd
kubectl delete networkpolicy argocd-redis-network-policy -n argocd
kubectl delete networkpolicy argocd-repo-server-network-policy -n argocd
kubectl delete networkpolicy argocd-server-network-policy -n argocd
```

!!! note "Network Policy Management"
    Consider managing network policies through GitOps once you understand your cluster's network requirements.

### Step 5: Apply Traefik CRDs

If using k3s, Traefik is pre-installed but may need CRDs for IngressRoute resources:

```bash
# Check if Traefik CRDs exist
kubectl get crd ingressroutes.traefik.io

# If not found, apply Traefik CRDs
kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v3.0/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml
```

### Step 6: Apply IngressRoutes

Apply the IngressRoute for ArgoCD:

```bash
kubectl apply -f argocd/bootstrap/ingress/
```

Verify the IngressRoute is created:

```bash
kubectl get ingressroute -n argocd
```

### Step 7: Deploy Cloudflared

Apply the cloudflared deployment:

```bash
kubectl apply -f base/infrastructure/cloudflare-tunnel/deployment.yaml
```

Check that cloudflared pods are running:

```bash
kubectl get pods -n cloudflare
kubectl logs -n cloudflare -l app=cloudflared
```

### Step 8: Verify Deployment

```bash
# Check ArgoCD pods
kubectl get pods -n argocd

# Get the admin password
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d && echo

# Port forward if needed (before tunnel works)
kubectl port-forward svc/argocd-server -n argocd 8080:443
```

Access ArgoCD at:

- Via port-forward: `https://localhost:8080`
- Via Cloudflare Tunnel: `https://dev.holm.chat/argoCD`
- Via local network: `http://192.168.8.197:30190/argoCD`

## Local Network Access

All services are accessible locally via Traefik's NodePort (30190) without requiring Cloudflare:

| Service | Local URL |
|---------|-----------|
| ArgoCD | `http://192.168.8.197:30190/argoCD` |
| Traefik Dashboard | `http://192.168.8.197:30190/traefik` |
| Argo Workflows | `http://192.168.8.197:30190/argo-workflows` |
| Velero/Backup | `http://192.168.8.197:30190/backup` |

IngressRoutes use PathPrefix-only matching (no Host requirement), enabling both external access via Cloudflare and local access via NodePort.

---

## Troubleshooting

### ConfigMap Not Found Errors

**Symptom:** ArgoCD components fail to start with errors about missing ConfigMaps:

```
Error: configmap "argocd-cm" not found
```

**Cause:** Kustomize patches require the ConfigMap to have specific labels (`app.kubernetes.io/part-of: argocd`) to match.

**Solution:**

```bash
# Check labels on the ConfigMap
kubectl get configmap argocd-cm -n argocd -o yaml | grep -A5 labels

# Add the required label
kubectl patch configmap argocd-cm -n argocd \
  --type merge \
  -p '{"metadata":{"labels":{"app.kubernetes.io/part-of":"argocd"}}}'

# Restart ArgoCD components to pick up changes
kubectl rollout restart deployment -n argocd
```

### Network Policies Blocking Access

**Symptom:** Cannot access ArgoCD UI, connection times out or is refused.

**Cause:** ArgoCD's default network policies are restrictive and may not allow ingress from Traefik or your network.

**Diagnosis:**

```bash
# List network policies
kubectl get networkpolicy -n argocd

# Check events for blocked traffic
kubectl get events -n argocd --sort-by='.lastTimestamp'

# Test connectivity from Traefik namespace
kubectl run -n traefik test-curl --rm -it --image=curlimages/curl -- \
  curl -k https://argocd-server.argocd.svc.cluster.local
```

**Solution:**

```bash
# Option 1: Delete all ArgoCD network policies
kubectl delete networkpolicy -n argocd --all

# Option 2: Create a policy that allows Traefik ingress
cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-traefik-ingress
  namespace: argocd
spec:
  podSelector: {}
  policyTypes:
    - Ingress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: traefik
EOF
```

### Cloudflared Probe Port Issues

**Symptom:** Cloudflared pods crash with probe failures:

```
Readiness probe failed: Get "http://10.x.x.x:2000/ready": dial tcp 10.x.x.x:2000: connect: connection refused
```

**Cause:** The cloudflared metrics/health port changed. Older documentation references port 2000, but the correct port is **20241**.

**Solution:**

Update the deployment to use port 20241:

```yaml
livenessProbe:
  httpGet:
    path: /ready
    port: 20241  # NOT 2000
  initialDelaySeconds: 10
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /ready
    port: 20241  # NOT 2000
  initialDelaySeconds: 5
  periodSeconds: 5
```

Verify:

```bash
# Check the port cloudflared is listening on
kubectl exec -n cloudflare deploy/cloudflared -- netstat -tlnp 2>/dev/null || \
kubectl exec -n cloudflare deploy/cloudflared -- ss -tlnp
```

### Using Existing k3s Traefik vs Deploying New One

**Symptom:** Confusion about whether to use k3s's built-in Traefik or deploy a separate instance.

**Background:** k3s ships with Traefik pre-installed in the `kube-system` namespace. Deploying another Traefik creates conflicts.

**Diagnosis:**

```bash
# Check for existing Traefik
kubectl get pods -A | grep traefik
kubectl get svc -A | grep traefik

# Check k3s Traefik service
kubectl get svc traefik -n kube-system -o yaml
```

**Options:**

=== "Use k3s Traefik (Recommended)"

    Point cloudflared to the k3s Traefik service:

    ```yaml
    # In Cloudflare Dashboard tunnel config
    Service: http://traefik.kube-system.svc.cluster.local:80
    ```

    Benefits:

    - No resource duplication
    - Automatic updates via k3s
    - Already integrated with k3s networking

=== "Deploy Separate Traefik"

    If you need specific Traefik features or versions:

    ```bash
    # Disable k3s Traefik first
    # In /etc/rancher/k3s/config.yaml:
    # disable:
    #   - traefik

    # Then deploy your own
    kubectl apply -k base/infrastructure/traefik/
    ```

    Point cloudflared to your Traefik:

    ```yaml
    Service: http://traefik.traefik.svc.cluster.local:80
    ```

**Verify routing works:**

```bash
# Test from within the cluster
kubectl run -n cloudflare test-curl --rm -it --image=curlimages/curl -- \
  curl -v http://traefik.kube-system.svc.cluster.local/ping
```

### ArgoCD Application Stuck Syncing

**Symptom:** Applications show "Syncing" or "Progressing" indefinitely.

**Diagnosis:**

```bash
# Check application status
kubectl get applications -n argocd

# Get detailed status
kubectl describe application <app-name> -n argocd

# Check for failed resources
kubectl get events -A --sort-by='.lastTimestamp' | tail -30
```

**Common fixes:**

```bash
# Force refresh
kubectl patch application <app-name> -n argocd \
  --type merge \
  -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'

# Or delete and let it recreate
kubectl delete application <app-name> -n argocd
# ArgoCD will recreate it from app-of-apps
```

### 502 Bad Gateway

**Symptom:** Accessing ArgoCD through Cloudflare returns 502.

**Cause:** The request reaches cloudflared and Traefik but the backend service is unreachable.

**Diagnosis:**

```bash
# Check ArgoCD server is running
kubectl get pods -n argocd -l app.kubernetes.io/name=argocd-server

# Check service exists
kubectl get svc argocd-server -n argocd

# Check endpoints are populated
kubectl get endpoints argocd-server -n argocd

# Test from Traefik's perspective
kubectl exec -n kube-system deploy/traefik -- \
  wget -qO- --no-check-certificate https://argocd-server.argocd.svc.cluster.local
```

**Solutions:**

```bash
# If pods not running, check events
kubectl describe pod -n argocd -l app.kubernetes.io/name=argocd-server

# Restart ArgoCD
kubectl rollout restart deployment argocd-server -n argocd
```

---

## Quick Reference Commands

```bash
# Full status check
kubectl get pods -n argocd && \
kubectl get pods -n cloudflare && \
kubectl get ingressroute -A

# Get ArgoCD password
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d && echo

# Watch all pods
kubectl get pods -A -w

# Check tunnel connectivity
kubectl logs -n cloudflare -l app=cloudflared --tail=50

# Test ArgoCD endpoint internally
kubectl run test --rm -it --image=curlimages/curl -- \
  curl -k https://argocd-server.argocd.svc.cluster.local
```
