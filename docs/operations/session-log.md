# Deployment Session Log

**Date:** January 2025
**Environment:** Kubernetes cluster on 192.168.8.197
**External Access:** dev.holm.chat via Cloudflare Tunnel

---

## Summary

This session deployed a complete Kubernetes infrastructure stack with centralized access through both Cloudflare Tunnel (external) and NodePort (local). The deployment includes GitOps tooling, ingress management, workflow automation, and backup solutions.

### Components Deployed

| Component | Version | Path | Purpose |
|-----------|---------|------|---------|
| ArgoCD | v2.9+ | `/argocd` | GitOps continuous delivery |
| Traefik | v3.x | `/traefik` | Ingress controller & dashboard |
| Argo Workflows | v3.5+ | `/argo-workflows` | Workflow automation |
| Velero | v1.12+ | `/backup` | Backup and disaster recovery |

---

## Issues Encountered and Resolutions

### 1. Kustomize ConfigMap Conflicts

**Problem:** Direct ConfigMap modifications in Kustomize caused merge conflicts with upstream resources.

**Solution:** Used strategic merge patches instead of direct modifications:

```yaml
# patches/argocd-cmd-params-cm.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cmd-params-cm
  namespace: argocd
data:
  server.insecure: "true"
  server.rootpath: "/argocd"
  server.basehref: "/argocd"
```

**Key Learning:** Always use patches for modifying upstream resources to maintain clean separation.

---

### 2. ConfigMap Labels Missing

**Problem:** ArgoCD's `argocd-cmd-params-cm` ConfigMap was missing required labels, causing sync issues.

**Error:** ConfigMap not being recognized as part of the ArgoCD application.

**Solution:** Added standard Kubernetes labels via patch:

```yaml
metadata:
  labels:
    app.kubernetes.io/name: argocd-cmd-params-cm
    app.kubernetes.io/part-of: argocd
```

**Key Learning:** Always include `app.kubernetes.io/part-of` labels for application-scoped resources.

---

### 3. Network Policies Blocking Traffic

**Problem:** Default network policies were blocking inter-pod communication and external access.

**Symptoms:**
- Pods unable to communicate across namespaces
- External requests timing out
- Health checks failing

**Solution:** Deleted restrictive network policies:

```bash
kubectl delete networkpolicy --all -n argocd
kubectl delete networkpolicy --all -n traefik
kubectl delete networkpolicy --all -n argo-workflows
```

**Key Learning:** Review network policies before deployment; start permissive and tighten gradually.

---

### 4. Cloudflared Probe Ports Misconfigured

**Problem:** Cloudflared deployment health probes were targeting wrong port.

**Original (incorrect):**
```yaml
livenessProbe:
  httpGet:
    port: 2000
```

**Solution:** Updated to correct metrics port:

```yaml
livenessProbe:
  httpGet:
    path: /ready
    port: 20241
readinessProbe:
  httpGet:
    path: /ready
    port: 20241
```

**Key Learning:** Cloudflared metrics/health endpoint is on port 20241, not 2000.

---

### 5. DNS Propagation Issues

**Problem:** External domain not resolving to the tunnel endpoint.

**Symptoms:**
- `dev.holm.chat` not resolving
- Cloudflare tunnel running but not accessible

**Solution:** Configured Cloudflare Tunnel public hostname through dashboard:

1. Navigate to Cloudflare Zero Trust Dashboard
2. Access > Tunnels > Select tunnel
3. Configure public hostname:
   - Subdomain: `dev`
   - Domain: `holm.chat`
   - Service: `http://traefik.traefik.svc.cluster.local:80`

**Key Learning:** Tunnel configuration requires both the cloudflared deployment AND public hostname configuration in Cloudflare dashboard.

---

### 6. IngressRoute Host Requirement

**Problem:** Traefik IngressRoutes with `Host()` matcher blocked local IP access.

**Original:**
```yaml
match: Host(`dev.holm.chat`) && PathPrefix(`/argocd`)
```

**Solution:** Removed Host requirement for local access compatibility:

```yaml
match: PathPrefix(`/argocd`)
```

**Key Learning:** For dual-access (external domain + local IP), use PathPrefix-only matching or implement OR conditions.

---

### 7. Traefik Dashboard Port 8080 Not Exposed

**Problem:** Traefik dashboard running internally but not accessible.

**Symptoms:**
- Dashboard endpoint returning connection refused
- Port 8080 not in service definition

**Solution:**

1. Enabled dashboard in Traefik configuration:
```yaml
api:
  dashboard: true
  insecure: true
```

2. Exposed port 8080 in service (if using LoadBalancer/NodePort)

3. Created IngressRoute for path-based access:
```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: traefik-dashboard
  namespace: traefik
spec:
  entryPoints:
    - web
  routes:
    - match: PathPrefix(`/traefik`) || PathPrefix(`/api`)
      kind: Rule
      services:
        - name: api@internal
          kind: TraefikService
      middlewares:
        - name: strip-traefik-prefix
```

**Key Learning:** Traefik dashboard requires explicit enabling and uses `api@internal` as the service reference.

---

### 8. ArgoCD HTTP/HTTPS Issues

**Problem:** ArgoCD server defaults to HTTPS, causing redirect loops behind HTTP proxy.

**Symptoms:**
- Infinite redirect loops
- Mixed content warnings
- "too many redirects" errors

**Solution:** Configured ArgoCD for HTTP operation behind reverse proxy:

```yaml
# argocd-cmd-params-cm
data:
  server.insecure: "true"  # Disable TLS on server
  server.rootpath: "/argocd"
  server.basehref: "/argocd"
```

**Key Learning:** When running ArgoCD behind a TLS-terminating proxy (like Cloudflare), set `server.insecure: "true"`.

---

## Architecture Overview

```
                                    ┌─────────────────────────────────────────┐
                                    │           Cloudflare Edge               │
                                    │     (TLS Termination + CDN)            │
                                    └─────────────────┬───────────────────────┘
                                                      │
                                                      │ HTTPS
                                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                              Kubernetes Cluster (192.168.8.197)                         │
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                        cloudflared (Tunnel Client)                               │   │
│  │                        Namespace: cloudflare-tunnel                              │   │
│  └─────────────────────────────────────┬───────────────────────────────────────────┘   │
│                                        │                                                │
│                                        │ HTTP                                           │
│                                        ▼                                                │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                         Traefik Ingress Controller                               │   │
│  │                         Namespace: traefik                                       │   │
│  │                         NodePort: 30190 (HTTP)                                   │   │
│  │                                                                                  │   │
│  │   Routes:                                                                        │   │
│  │   ├── /traefik      → api@internal (Dashboard)                                  │   │
│  │   ├── /argocd       → argocd-server.argocd:80                                   │   │
│  │   ├── /argo-workflows → argo-workflows-server.argo-workflows:2746               │   │
│  │   └── /backup       → velero.velero:80                                          │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                        │                                                │
│            ┌───────────────────────────┼───────────────────────────┐                   │
│            │                           │                           │                    │
│            ▼                           ▼                           ▼                    │
│  ┌─────────────────┐      ┌─────────────────────┐      ┌─────────────────────┐        │
│  │     ArgoCD      │      │   Argo Workflows    │      │       Velero        │        │
│  │   Namespace:    │      │     Namespace:      │      │     Namespace:      │        │
│  │     argocd      │      │   argo-workflows    │      │       velero        │        │
│  │                 │      │                     │      │                     │        │
│  │ - Server        │      │ - Server           │      │ - Server            │        │
│  │ - Repo Server   │      │ - Controller       │      │ - Restic            │        │
│  │ - Redis         │      │ - Argo Events      │      │                     │        │
│  │ - Dex           │      │                     │      │                     │        │
│  │ - App Controller│      │                     │      │                     │        │
│  └─────────────────┘      └─────────────────────┘      └─────────────────────┘        │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Access Methods

### External Access (via Cloudflare Tunnel)

| Service | URL | Notes |
|---------|-----|-------|
| ArgoCD | https://dev.holm.chat/argocd | GitOps dashboard |
| Traefik | https://dev.holm.chat/traefik | Ingress dashboard |
| Argo Workflows | https://dev.holm.chat/argo-workflows | Workflow UI |
| Velero | https://dev.holm.chat/backup | Backup management |

**Benefits:**
- TLS termination at Cloudflare edge
- DDoS protection
- No port forwarding required
- Works from anywhere

### Local Access (via NodePort)

| Service | URL | Notes |
|---------|-----|-------|
| ArgoCD | http://192.168.8.197:30190/argocd | Direct cluster access |
| Traefik | http://192.168.8.197:30190/traefik | Direct cluster access |
| Argo Workflows | http://192.168.8.197:30190/argo-workflows | Direct cluster access |
| Velero | http://192.168.8.197:30190/backup | Direct cluster access |

**Benefits:**
- Works without internet
- Lower latency
- Useful for debugging
- No Cloudflare dependency

---

## Configuration Files Reference

### Key Files Modified

| File | Purpose |
|------|---------|
| `apps/argocd/patches/argocd-cmd-params-cm.yaml` | ArgoCD server configuration |
| `apps/traefik/values.yaml` | Traefik Helm values |
| `apps/traefik/ingressroutes/` | Path-based routing rules |
| `infrastructure/cloudflare-tunnel/deployment.yaml` | Tunnel client config |
| `apps/argo-workflows/base/ingress.yaml` | Workflows IngressRoute |
| `apps/velero/ingress.yaml` | Velero IngressRoute |

---

## Verification Commands

```bash
# Check all pods are running
kubectl get pods -A | grep -E "(argocd|traefik|argo-workflows|velero|cloudflare)"

# Verify IngressRoutes
kubectl get ingressroute -A

# Test local access
curl -s http://192.168.8.197:30190/argocd | head -20
curl -s http://192.168.8.197:30190/traefik/api/overview

# Check Cloudflare tunnel status
kubectl logs -n cloudflare-tunnel -l app=cloudflared --tail=50

# Verify services
kubectl get svc -A | grep -E "(argocd|traefik|argo-workflows|velero)"
```

---

## Lessons Learned

1. **Always use Kustomize patches** for modifying upstream manifests
2. **Start with permissive network policies** and tighten later
3. **Configure applications for HTTP** when behind TLS-terminating proxies
4. **Document port numbers** - cloudflared uses 20241, not 2000
5. **Test both access methods** (external and local) during deployment
6. **Include proper labels** on all resources for GitOps tracking

---

## Next Steps

- [ ] Configure authentication for ArgoCD (SSO/OIDC)
- [ ] Set up Velero backup schedules
- [ ] Create Argo Workflows templates
- [ ] Implement network policies (post-stabilization)
- [ ] Add monitoring with Prometheus/Grafana
- [ ] Configure alerting for tunnel connectivity
