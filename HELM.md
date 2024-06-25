# Helm

## Building the helm charts

```shell
go install sigs.k8s.io/kustomize/kustomize/v5@latest
go install github.com/arttor/helmify/cmd/helmify@latest

kustomize build config/default | helmify -cert-manager-as-subchart -cert-manager-version 1.15.0 -generate-defaults -image-pull-secrets helm-charts/kode
kustomize build config/crd | helmify helm-charts/kode-crd
```
