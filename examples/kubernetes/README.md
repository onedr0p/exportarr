# Kubernetes Examples

## Installation via Kustomize

To install via kustomize, you'll need to create a secret, and a kustomization:
```yaml
# secret.yaml
---
apiVersion: v1
kind: Secret
metadata:
    name: prowlarr-apikey
stringData:
    api-key: xxx
```

```yaml
# kustomization.yaml
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - https://github.com/onedr0p/exportarr//kubernetes/prowlarr-exporter?ref=v1.5.5
  - secret.yaml
images:
  - name: ghcr.io/onedr0p/exportarr
    newTag: v1
```

## Kustomizing...
To customize environmental variables, or anything else, leverage a [Strategic Merge Patch](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_patchesstrategicmerge_):

```yaml
# patch.yaml
---
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: prowlarr-exporter
  spec:
    template:
      spec:
        containers:
          - name: prowlarr-exporter
            env:
              - name: LOG_LEVEL
                value: debug
```

And then add the patch to your kustomization:
```yaml
# kustomization.yaml
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - https://github.com/onedr0p/exportarr//kubernetes/prowlarr-exporter?ref=v1.5.5
  - secret.yaml
images:
  - name: ghcr.io/onedr0p/exportarr
    newTag: v1

patchesStrategicMerge:
  - patch.yaml
```
