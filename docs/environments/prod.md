# Production Environment

The production environment requires manual approval for all deployments.

## Configuration

| Property | Value |
|----------|-------|
| Namespace | `prod` |
| Auto Sync | ❌ Disabled |
| Self Heal | ❌ Disabled |
| Prune | ❌ Disabled |

## Resource Quotas

```yaml
requests.cpu: "16"
requests.memory: 32Gi
limits.cpu: "32"
limits.memory: 64Gi
pods: "100"
```

## Manual Sync Required

Production deployments require explicit action:

=== "ArgoCD UI"

    1. Navigate to [dev.holm.chat/argoCD](https://dev.holm.chat/argoCD)
    2. Find `prod-infrastructure` application
    3. Click **Sync**
    4. Review changes
    5. Click **Synchronize**

=== "ArgoCD CLI"

    ```bash
    # Preview changes
    argocd app diff prod-infrastructure

    # Sync with confirmation
    argocd app sync prod-infrastructure
    ```

=== "kubectl"

    ```bash
    # Trigger sync via annotation
    kubectl -n argocd patch application prod-infrastructure \
      --type merge \
      -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'
    ```

## Why Manual Sync?

!!! danger "Production Protection"
    Automatic sync in production can cause:

    - Unplanned downtime
    - Breaking changes without review
    - Compliance violations
    - Difficult rollbacks

Manual sync ensures:

- Change review before deployment
- Scheduled maintenance windows
- Audit trail of who approved changes
- Time to prepare rollback plans

## Deployment Checklist

Before syncing to production:

- [ ] Changes tested in staging
- [ ] No failing tests
- [ ] Changelog updated
- [ ] Stakeholders notified
- [ ] Rollback plan ready
- [ ] Monitoring dashboards open
