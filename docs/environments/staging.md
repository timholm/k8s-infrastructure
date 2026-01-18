# Staging Environment

The staging environment mirrors production configuration for pre-release validation.

## Configuration

| Property | Value |
|----------|-------|
| Namespace | `staging` |
| Auto Sync | ✅ Enabled |
| Self Heal | ✅ Enabled |
| Prune | ✅ Enabled |

## Resource Quotas

```yaml
requests.cpu: "4"
requests.memory: 8Gi
limits.cpu: "8"
limits.memory: 16Gi
pods: "40"
```

## Purpose

Staging serves as the final validation before production:

- Integration testing
- Performance testing
- Security scanning
- User acceptance testing

## Kustomization

```yaml
# environments/staging/infrastructure/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: staging

resources:
  - ../../../base/infrastructure

commonLabels:
  environment: staging
```

## Behavior

- **Automatic sync**: Changes deploy automatically for testing
- **Production-like**: Same base configuration as prod
- **Lower limits**: Reduced quotas to save resources
