# K8s Infrastructure

Welcome to the documentation for the **k8s-infrastructure** GitOps repository.

This repository manages Kubernetes infrastructure using the GitOps methodology with ArgoCD.

## Quick Links

- [Architecture Overview](architecture/overview.md) - Understand how everything fits together
- [Bootstrap Guide](operations/bootstrap.md) - Get started deploying to your cluster
- [Adding Applications](operations/adding-apps.md) - Deploy new infrastructure components

## What is GitOps?

GitOps is an operational framework that applies DevOps best practices for infrastructure automation:

- **Git as the single source of truth** - All configuration is stored in Git
- **Declarative configuration** - Describe the desired state, not how to get there
- **Automated reconciliation** - ArgoCD ensures the cluster matches Git
- **Pull-based deployment** - Changes are pulled from Git, not pushed to the cluster

## Repository Structure

```
k8s-infrastructure/
├── argocd/                  # ArgoCD configuration
│   ├── bootstrap/           # Installation and app-of-apps
│   └── projects/            # ArgoCD Application definitions
├── base/                    # Base manifests (shared across environments)
│   └── infrastructure/      # Core infrastructure components
├── environments/            # Environment-specific overlays
│   ├── dev/
│   ├── staging/
│   └── prod/
└── docs/                    # This documentation
```

## Key Components

| Component | Purpose | URL |
|-----------|---------|-----|
| ArgoCD | GitOps continuous delivery | [dev.holm.chat/argo](https://dev.holm.chat/argo) |
| Traefik | Ingress controller & reverse proxy | Internal |
| Cloudflare Tunnel | Secure external access | - |
