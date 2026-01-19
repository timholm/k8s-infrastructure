# Bootstrap Guide

This guide walks through deploying the entire infrastructure from scratch.

## Prerequisites

- Kubernetes cluster (any: kind, k3s, EKS, GKE, AKS)
- `kubectl` configured with cluster access
- `argocd` CLI (optional, for management)
- Cloudflare account with Tunnel token

## Step 1: Create Cloudflare Tunnel Secret

Before deploying, create the tunnel secret:

```bash
# Create namespace
kubectl create namespace cloudflare

# Create secret with your tunnel token
kubectl create secret generic cloudflare-tunnel-token \
  --from-literal=token=<your-tunnel-token> \
  -n cloudflare
```

!!! warning "Get your token first"
    Create a tunnel in [Cloudflare Zero Trust Dashboard](https://one.dash.cloudflare.com/) and copy the token.

## Step 2: Bootstrap ArgoCD

Deploy ArgoCD and the app-of-apps:

```bash
# Apply the bootstrap kustomization
kubectl apply -k argocd/bootstrap/
```

This installs:

- ArgoCD (from upstream manifests)
- Custom ConfigMaps (RBAC, settings)
- App-of-apps (manages all other Applications)
- Traefik IngressRoute for ArgoCD

## Step 3: Wait for ArgoCD

```bash
# Watch pods come up
kubectl get pods -n argocd -w

# Wait for all deployments
kubectl rollout status deployment -n argocd --timeout=300s
```

## Step 4: Access ArgoCD

### Get Admin Password

```bash
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d && echo
```

### Via Port Forward (Before Tunnel Works)

```bash
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Open https://localhost:8080
```

### Via Cloudflare Tunnel

Once the tunnel is connected:

- URL: [https://dev.holm.chat/argoCD](https://dev.holm.chat/argoCD)
- Username: `admin`
- Password: (from secret above)

## Step 5: Verify Deployment

In ArgoCD UI, you should see:

| Application | Status |
|-------------|--------|
| app-of-apps | Synced ✅ |
| dev-infrastructure | Synced ✅ |
| staging-infrastructure | Synced ✅ |
| prod-infrastructure | OutOfSync (manual sync required) |

## Step 6: Configure Cloudflare Tunnel Routes

In Cloudflare Dashboard, configure the tunnel:

| Hostname | Service |
|----------|---------|
| `dev.holm.chat/*` | `http://traefik.traefik.svc.cluster.local:80` |
| `docs.holm.chat/*` | `http://traefik.traefik.svc.cluster.local:80` |

## Troubleshooting

### ArgoCD Pods Not Starting

```bash
# Check events
kubectl get events -n argocd --sort-by='.lastTimestamp'

# Check pod logs
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-server
```

### Tunnel Not Connecting

```bash
# Check cloudflared logs
kubectl logs -n cloudflare -l app=cloudflared

# Verify secret exists
kubectl get secret cloudflare-tunnel-token -n cloudflare
```

### 502 Bad Gateway

Traefik or the target service isn't running:

```bash
# Check Traefik
kubectl get pods -n traefik

# Check ArgoCD
kubectl get pods -n argocd
```
