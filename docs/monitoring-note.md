make kind-localhost-setup
make helm-install-all ENV=localhost

make kind-dev-setup
make helm-install-all ENV=dev

```

# 클러스터 ConfigMap 확인
kubectl get configmap cluster-info -n kube-system -o yaml

# 또는 노드 라벨 확인
kubectl get nodes --show-labels

# 또는 namespace annotation 확인
kubectl get ns wealist-dev -o yaml

```
