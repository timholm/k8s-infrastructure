# k8s-infrastructure

GitOps repository for Kubernetes infrastructure managed by ArgoCD.

## Repository Structure

```
.
├── argocd/
│   ├── bootstrap/          # ArgoCD installation and app-of-apps
│   └── projects/           # ArgoCD project definitions
├── base/
│   ├── apps/               # Base application manifests
│   └── infrastructure/     # Base infrastructure components
└── environments/
    ├── dev/                # Development environment
    ├── staging/            # Staging environment
    └── prod/               # Production environment
```

## Getting Started

### Prerequisites

- Kubernetes cluster
- `kubectl` configured
- `argocd` CLI (optional)

### Bootstrap ArgoCD

1. Install ArgoCD in your cluster:

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f argocd/bootstrap/install.yaml
```

2. Deploy the app-of-apps to bootstrap all applications:

```bash
kubectl apply -f argocd/bootstrap/app-of-apps.yaml
```

### Access ArgoCD UI

```bash
# Get the initial admin password
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d

# Port-forward to access the UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
```

Then open https://localhost:8080 and login with username `admin`.

## Environment Promotion

Changes flow through environments:
1. **dev** - Development and testing
2. **staging** - Pre-production validation
3. **prod** - Production deployment

Each environment uses Kustomize overlays on top of base manifests.

## Adding New Infrastructure

1. Add base manifests to `base/infrastructure/<component>/`
2. Create kustomization.yaml referencing the base
3. Add environment-specific patches in `environments/<env>/infrastructure/`
4. Update the ArgoCD Application in `argocd/projects/`
