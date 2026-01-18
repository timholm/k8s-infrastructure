# Troubleshooting

Common issues and their solutions.

## ArgoCD Issues

### Application Stuck in "Progressing"

```bash
# Check application status
argocd app get <app-name>

# Check for failed resources
kubectl get events -n <namespace> --sort-by='.lastTimestamp'

# Force refresh
argocd app refresh <app-name> --hard
```

### Sync Failed

```bash
# View sync details
argocd app sync <app-name> --dry-run

# Check diff
argocd app diff <app-name>

# Force sync (use carefully)
argocd app sync <app-name> --force
```

### Can't Access ArgoCD UI

```bash
# Check pods
kubectl get pods -n argocd

# Port forward directly
kubectl port-forward svc/argocd-server -n argocd 8080:443

# Check ingress
kubectl get ingressroute -n argocd
```

## Traefik Issues

### 404 Not Found

IngressRoute not matching:

```bash
# List all IngressRoutes
kubectl get ingressroute -A

# Check Traefik logs
kubectl logs -n traefik -l app.kubernetes.io/name=traefik
```

### 502 Bad Gateway

Backend service not running:

```bash
# Check if target service exists
kubectl get svc -A | grep <service-name>

# Check if pods are running
kubectl get pods -n <namespace>

# Check endpoints
kubectl get endpoints -n <namespace> <service-name>
```

### Middleware Not Applied

```bash
# Verify middleware exists in same namespace or use cross-namespace reference
kubectl get middleware -A

# Check IngressRoute references correct middleware
kubectl get ingressroute <name> -n <namespace> -o yaml
```

## Cloudflare Tunnel Issues

### Tunnel Disconnected

```bash
# Check cloudflared pods
kubectl get pods -n cloudflare

# Check logs
kubectl logs -n cloudflare -l app=cloudflared --tail=100

# Verify secret
kubectl get secret cloudflare-tunnel-token -n cloudflare
```

### ERR_CONNECTION_REFUSED

Tunnel can't reach Traefik:

```bash
# Verify Traefik service
kubectl get svc -n traefik

# Test connectivity from cloudflared pod
kubectl exec -n cloudflare deploy/cloudflared -- \
  wget -qO- http://traefik.traefik.svc.cluster.local/ping
```

## Kustomize Issues

### Build Fails

```bash
# Validate kustomization
kubectl kustomize environments/dev/infrastructure/

# Check for missing resources
kubectl kustomize --enable-alpha-plugins environments/dev/infrastructure/
```

### Patches Not Applying

Ensure target selectors match:

```yaml
patches:
  - patch: |-
      ...
    target:
      kind: Deployment
      name: exact-name-here  # Must match exactly
```

## General Kubernetes Issues

### Pods in CrashLoopBackOff

```bash
# Check logs
kubectl logs <pod-name> -n <namespace> --previous

# Describe pod
kubectl describe pod <pod-name> -n <namespace>
```

### ImagePullBackOff

```bash
# Check image name
kubectl describe pod <pod-name> -n <namespace> | grep Image

# Check image pull secrets
kubectl get secrets -n <namespace>
```

### Resource Quota Exceeded

```bash
# Check quota usage
kubectl describe resourcequota -n <namespace>

# List resource requests
kubectl top pods -n <namespace>
```

## Quick Diagnostics Script

```bash
#!/bin/bash
# Save as diagnose.sh

echo "=== ArgoCD Status ==="
kubectl get pods -n argocd

echo -e "\n=== Traefik Status ==="
kubectl get pods -n traefik

echo -e "\n=== Cloudflare Tunnel Status ==="
kubectl get pods -n cloudflare

echo -e "\n=== Recent Events ==="
kubectl get events -A --sort-by='.lastTimestamp' | tail -20

echo -e "\n=== ArgoCD Applications ==="
kubectl get applications -n argocd
```
