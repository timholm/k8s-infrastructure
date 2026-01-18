# k8s-infrastructure

GitOps repository for Kubernetes infrastructure managed by ArgoCD.

| Service | URL |
|---------|-----|
| Documentation | [docs.holm.chat](https://docs.holm.chat) |
| ArgoCD | [dev.holm.chat/argo](https://dev.holm.chat/argo) |

## Quick Start

```bash
# 1. Create Cloudflare Tunnel secret
kubectl create namespace cloudflare
kubectl create secret generic cloudflare-tunnel-token \
  --from-literal=token=<your-token> -n cloudflare

# 2. Bootstrap everything
kubectl apply -k argocd/bootstrap/
```

See the [full bootstrap guide](https://docs.holm.chat/operations/bootstrap/) for details.

## Repository Structure

```
.
├── argocd/
│   ├── bootstrap/          # ArgoCD installation and app-of-apps
│   │   └── ingress/        # Traefik IngressRoute for ArgoCD
│   └── projects/           # ArgoCD Applications per environment
├── base/
│   └── infrastructure/     # Base infrastructure components
│       ├── traefik/        # Ingress controller
│       ├── cloudflare-tunnel/  # External access
│       ├── namespace/      # Resource quotas & limits
│       └── network-policies/
├── environments/
│   ├── dev/                # Auto-sync enabled
│   ├── staging/            # Auto-sync enabled
│   └── prod/               # Manual sync required
└── docs/                   # MkDocs documentation
```

## Components

| Component | Purpose |
|-----------|---------|
| **ArgoCD** | GitOps continuous delivery |
| **Traefik** | Ingress controller & reverse proxy |
| **Cloudflare Tunnel** | Secure external access (no public IPs) |

## Documentation

Full documentation is available at [docs.holm.chat](https://docs.holm.chat):

- [Architecture Overview](https://docs.holm.chat/architecture/overview/)
- [Kubernetes DNS Explained](https://docs.holm.chat/architecture/kubernetes-dns/) - Why `traefik.traefik`?
- [Traffic Flow](https://docs.holm.chat/architecture/traffic-flow/)
- [Bootstrap Guide](https://docs.holm.chat/operations/bootstrap/)
- [Adding Applications](https://docs.holm.chat/operations/adding-apps/)
- [Troubleshooting](https://docs.holm.chat/operations/troubleshooting/)

## Environment Promotion

| Environment | Namespace | Auto Sync | Notes |
|-------------|-----------|-----------|-------|
| dev | `dev` | ✅ | Fast feedback |
| staging | `staging` | ✅ | Pre-prod validation |
| prod | `prod` | ❌ | Manual approval required |

## Local Docs Development

```bash
pip install mkdocs-material pymdown-extensions
mkdocs serve
# Open http://127.0.0.1:8000
```
