# Traffic Flow

This document details how traffic flows from the internet to your applications.

## External Access via Cloudflare Tunnel

```mermaid
sequenceDiagram
    participant User
    participant Cloudflare
    participant Tunnel as cloudflared
    participant Traefik
    participant ArgoCD

    User->>Cloudflare: GET https://dev.holm.chat/argoCD
    Cloudflare->>Tunnel: Forward via Tunnel
    Tunnel->>Traefik: http://traefik.traefik:80/argoCD
    Note over Traefik: Match IngressRoute<br/>Strip /argoCD prefix
    Traefik->>ArgoCD: https://argocd-server:443/
    ArgoCD->>Traefik: Response
    Traefik->>Tunnel: Response
    Tunnel->>Cloudflare: Response
    Cloudflare->>User: Response
```

## Why This Architecture?

### No Public IPs Required

Traditional Kubernetes ingress requires:
- A LoadBalancer service (costs money)
- Public IP addresses
- Firewall configuration
- DDoS protection

With Cloudflare Tunnel:
- **Zero public IPs** - Tunnel creates outbound connection
- **Free DDoS protection** - Cloudflare handles it
- **Built-in TLS** - Cloudflare manages certificates
- **Zero attack surface** - No inbound ports open

### Path-Based Routing

Multiple services share one domain:

| URL | Service |
|-----|---------|
| `dev.holm.chat/argoCD` | ArgoCD |
| `dev.holm.chat/traefik` | Traefik Dashboard |
| `dev.holm.chat/` | Default application |

## Traefik IngressRoute

Traefik uses `IngressRoute` CRDs for advanced routing:

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: argocd-server
  namespace: argocd
spec:
  entryPoints:
    - websecure
  routes:
    - match: Host(`dev.holm.chat`) && PathPrefix(`/argo`)
      kind: Rule
      services:
        - name: argocd-server
          port: 443
      middlewares:
        - name: argocd-stripprefix  # Remove /argo before forwarding
```

### Middleware Chain

```mermaid
graph LR
    Request["/argo/applications"] --> Strip[stripPrefix<br/>Remove /argo]
    Strip --> Headers[headers<br/>Add X-Forwarded-Proto]
    Headers --> Service["/applications"]
```

## Internal Traffic

Services communicate directly within the cluster:

```mermaid
graph LR
    subgraph Cluster Network
        ArgoCD -->|Git sync| GitHub[GitHub API]
        ArgoCD -->|Deploy| K8sAPI[Kubernetes API]
        Apps -->|Metrics| Prometheus
    end
```

Internal traffic bypasses Traefik and Cloudflare entirely.
