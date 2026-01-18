# Adding Applications

This guide explains how to add new infrastructure components to the GitOps repository.

## Overview

Adding a new component involves:

1. Creating base manifests
2. Adding environment overlays
3. Creating an ArgoCD Application
4. Committing and pushing

## Step 1: Create Base Manifests

Create a new directory under `base/infrastructure/`:

```bash
mkdir -p base/infrastructure/my-app
```

Add your Kubernetes manifests:

```yaml
# base/infrastructure/my-app/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
        - name: my-app
          image: my-app:latest
          ports:
            - containerPort: 8080
```

Create the kustomization file:

```yaml
# base/infrastructure/my-app/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - deployment.yaml
  - service.yaml
```

## Step 2: Add to Base Infrastructure

Update the base kustomization:

```yaml
# base/infrastructure/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - namespace
  - network-policies
  - traefik
  - cloudflare-tunnel
  - my-app  # Add your new app
```

## Step 3: Environment Overlays (Optional)

If you need environment-specific configuration:

```yaml
# environments/dev/infrastructure/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: dev

resources:
  - ../../../base/infrastructure

patches:
  # Dev-specific patch for my-app
  - patch: |-
      - op: replace
        path: /spec/replicas
        value: 1
    target:
      kind: Deployment
      name: my-app
```

## Step 4: Test Locally

Validate your manifests:

```bash
# Build and review the output
kubectl kustomize environments/dev/infrastructure/

# Dry-run apply
kubectl apply -k environments/dev/infrastructure/ --dry-run=client
```

## Step 5: Commit and Push

```bash
git add -A
git commit -m "Add my-app infrastructure component"
git push
```

## Step 6: Verify in ArgoCD

ArgoCD will automatically detect the change:

1. Open [dev.holm.chat/argo](https://dev.holm.chat/argo)
2. Find `dev-infrastructure` application
3. Watch it sync automatically (or click Sync for prod)

## Adding External Ingress

If your app needs external access, add a Traefik IngressRoute:

```yaml
# base/infrastructure/my-app/ingressroute.yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: my-app
spec:
  entryPoints:
    - websecure
  routes:
    - match: Host(`dev.holm.chat`) && PathPrefix(`/my-app`)
      kind: Rule
      services:
        - name: my-app
          port: 8080
      middlewares:
        - name: my-app-stripprefix
---
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: my-app-stripprefix
spec:
  stripPrefix:
    prefixes:
      - /my-app
```

Then configure the route in Cloudflare Tunnel dashboard.

## Using Helm Charts

For Helm-based applications, use ArgoCD's Helm support:

```yaml
# argocd/projects/my-helm-app.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: prometheus
  namespace: argocd
spec:
  project: infrastructure
  source:
    repoURL: https://prometheus-community.github.io/helm-charts
    chart: prometheus
    targetRevision: 25.0.0
    helm:
      values: |
        server:
          persistentVolume:
            enabled: false
  destination:
    server: https://kubernetes.default.svc
    namespace: monitoring
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```
