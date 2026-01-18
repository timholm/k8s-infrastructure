# Development Environment

The development environment is used for rapid iteration and testing.

## Configuration

| Property | Value |
|----------|-------|
| Namespace | `dev` |
| Auto Sync | ✅ Enabled |
| Self Heal | ✅ Enabled |
| Prune | ✅ Enabled |

## Resource Quotas

```yaml
requests.cpu: "2"
requests.memory: 4Gi
limits.cpu: "4"
limits.memory: 8Gi
pods: "25"
```

## Kustomization

```yaml
# environments/dev/infrastructure/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: dev

resources:
  - ../../../base/infrastructure

commonLabels:
  environment: dev

patches:
  # Lower resource quotas for dev
  - patch: |-
      - op: replace
        path: /spec/hard/requests.cpu
        value: "2"
      # ... more patches
    target:
      kind: ResourceQuota
```

## Behavior

- **Automatic sync**: Changes merged to `main` deploy immediately
- **Self-healing**: Manual changes in cluster are reverted
- **Pruning**: Deleted resources in Git are removed from cluster

!!! tip "Fast Feedback"
    Dev environment is designed for fast feedback. Break things here, not in prod!
