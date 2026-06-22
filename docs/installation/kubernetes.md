# Kubernetes Installation

## Prerequisites

- Kubernetes 1.24+
- kubectl configured
- Ingress controller (nginx-ingress recommended)
- cert-manager (optional, for TLS)

## Quick Install

```bash
# Apply all manifests
kubectl apply -k deploy/k8s/

# Verify
kubectl get pods -n x-ui-pro
kubectl get svc -n x-ui-pro
```

## Access

After deploying with the default Ingress:

```
https://x-ui-pro.example.com/panel/
```

Update the domain in `deploy/k8s/ingress.yaml` and the TLS secret name.

## Customization

### Resource Limits

Edit `deploy/k8s/deployment.yaml`:

```yaml
resources:
  requests:
    cpu: "200m"
    memory: "256Mi"
  limits:
    cpu: "1"
    memory: "1Gi"
```

### Storage

Default PVC uses `standard` storage class. Change in `deploy/k8s/pvc.yaml`:

```yaml
spec:
  storageClassName: ssd
```

### Ingress TLS

1. Install cert-manager
2. Update `deploy/k8s/ingress.yaml` annotations if needed
3. Replace `x-ui-pro.example.com` with your domain

## Scaling

HPA is configured to scale between 1-5 replicas at 70% CPU or 80% memory:

```bash
kubectl get hpa -n x-ui-pro
```

Note: For multi-replica setups, use PostgreSQL backend instead of SQLite.

## Uninstall

```bash
kubectl delete -k deploy/k8s/
```

## Manifests Structure

```
deploy/k8s/
├── namespace.yaml     # x-ui-pro namespace
├── configmap.yaml     # Non-sensitive configuration
├── secret.yaml        # Sensitive configuration (JWT, passwords)
├── pvc.yaml           # PersistentVolumeClaim for data + certs
├── deployment.yaml    # Main deployment with health probes
├── service.yaml       # ClusterIP service
├── ingress.yaml       # TLS ingress with nginx annotations
├── hpa.yaml           # HorizontalPodAutoscaler
└── kustomization.yaml # Kustomize entry point
```
