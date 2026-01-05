# Prod Compute

프로덕션 EKS 클러스터와 관련 리소스입니다.

## 개요

이 레이어는 Kubernetes 워크로드를 실행하기 위한 EKS 클러스터, 노드 그룹, Add-on, Pod Identity를 관리합니다.

**의존성**: `prod/foundation`이 먼저 배포되어야 합니다.

## 구성 요소

| 리소스 | 설명 |
|--------|------|
| EKS Cluster | Kubernetes 컨트롤 플레인 |
| Managed Node Group | Spot 인스턴스 워커 노드 |
| EKS Add-ons | VPC CNI, CoreDNS, kube-proxy, EBS CSI, Pod Identity Agent |
| Pod Identity | 서비스별 AWS 권한 |

## 아키텍처

```
┌─────────────────────────────────────────────────────────────────┐
│ EKS Cluster (wealist-prod-eks)                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Control Plane (AWS 관리형)                                │   │
│  │ - API Server                                             │   │
│  │ - etcd                                                   │   │
│  │ - Controller Manager                                     │   │
│  │ - Scheduler                                              │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Managed Node Group (Spot)                                │   │
│  │                                                          │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐          │   │
│  │  │ t3.medium  │  │ t3.medium  │  │ t3.medium  │          │   │
│  │  │ (Spot)     │  │ (Spot)     │  │ (Spot)     │          │   │
│  │  │ 50GB EBS   │  │ 50GB EBS   │  │ 50GB EBS   │          │   │
│  │  └────────────┘  └────────────┘  └────────────┘          │   │
│  │                                                          │   │
│  │  min: 2, desired: 3, max: 6                              │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Add-ons                                                  │   │
│  │ - vpc-cni (Istio Sidecar 지원)                            │   │
│  │ - coredns                                                │   │
│  │ - kube-proxy                                             │   │
│  │ - aws-ebs-csi-driver                                     │   │
│  │ - eks-pod-identity-agent                                 │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## EKS 클러스터 설정

| 설정 | 값 | 비고 |
|------|-----|------|
| Kubernetes 버전 | 1.30 | Istio 1.28.2 Sidecar 지원 |
| API 엔드포인트 | Public + Private | 외부/내부 접근 가능 |
| Secrets 암호화 | KMS | foundation 레이어 키 사용 |
| 로깅 | 5가지 전부 활성화 | api, audit, authenticator, controllerManager, scheduler |

## Istio Sidecar 지원

### VPC CNI 설정

```hcl
vpc-cni = {
  configuration_values = jsonencode({
    env = {
      POD_SECURITY_GROUP_ENFORCING_MODE = "standard"
      ENABLE_PREFIX_DELEGATION = "true"
      WARM_PREFIX_TARGET = "1"
    }
  })
}
```

### Security Group 규칙

| 포트 | 프로토콜 | 용도 |
|------|----------|------|
| 15001-15006 | TCP | Envoy Sidecar 트래픽 redirect |
| 15012 | TCP | XDS (istiod ↔ Sidecar 통신) |
| 15020-15021 | TCP | Metrics, readiness |
| 53 | TCP/UDP | CoreDNS |

## Node Group 설정

### Spot 인스턴스

| 설정 | 값 |
|------|-----|
| 인스턴스 타입 | t3.medium, t3a.medium, t3.large, t3a.large |
| Capacity | SPOT (100%) |
| 노드 수 | min: 2, desired: 3, max: 6 |
| 디스크 | 50GB EBS |

### Spot 중단 대비

다양한 인스턴스 타입을 지정하여 Spot 가용성을 확보합니다:

```hcl
instance_types = [
  "t3.medium",   # 기본
  "t3a.medium",  # AMD (더 저렴)
  "t3.large",    # Fallback (메모리 2배)
  "t3a.large"    # Fallback AMD
]
```

## Pod Identity 연결

| 서비스 | Namespace | ServiceAccount | AWS 권한 |
|--------|-----------|----------------|----------|
| ALB Controller | kube-system | aws-load-balancer-controller | ELB, EC2, ACM, WAF |
| External Secrets | external-secrets | external-secrets | Secrets Manager, SSM, KMS |
| External DNS | external-dns | external-dns | Route53 |
| cert-manager | cert-manager | cert-manager | Route53 (DNS-01) |
| storage-service | wealist-prod | storage-service | S3 |
| Cluster Autoscaler | kube-system | cluster-autoscaler | ASG, EC2 |

## 배포

### 사전 요구사항

1. `prod/foundation` 배포 완료
2. AWS CLI 설정
3. kubectl 설치

### 배포 명령

```bash
cd terraform/prod/compute

# 초기화
terraform init

# 계획 확인 (약 2분)
terraform plan

# 적용 (약 15-20분)
terraform apply
```

### 배포 소요 시간

| 리소스 | 예상 시간 |
|--------|----------|
| EKS Cluster | 10-12분 |
| Node Group | 3-5분 |
| Add-ons | 2-3분 |
| Pod Identity | 1분 |

## 배포 후 설정

### 1. kubeconfig 설정

```bash
aws eks update-kubeconfig --name wealist-prod-eks --region ap-northeast-2
```

### 2. Gateway API CRDs 설치

```bash
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml
```

### 3. Istio Sidecar 설치 (Terraform에서 자동 설치됨)

Terraform의 `helm-releases.tf`에서 Istio가 자동으로 설치됩니다:
- `istio-base`: Istio CRDs
- `istiod`: Control plane (Sidecar 모드, profile 설정 없음)

수동 설치가 필요한 경우:
```bash
# Istio CLI 설치 (1.28.2 - EKS 1.34 호환)
curl -L https://istio.io/downloadIstio | ISTIO_VERSION=1.28.2 sh -
export PATH=$PWD/istio-1.28.2/bin:$PATH

# Sidecar 모드 설치 (default profile)
istioctl install -y

# 설치 확인
kubectl get pods -n istio-system
istioctl proxy-status
```

### 4. 네임스페이스 설정 (Terraform에서 자동 생성됨)

Terraform의 `namespaces.tf`에서 네임스페이스가 자동으로 생성됩니다:
```hcl
resource "kubernetes_namespace" "wealist_prod" {
  metadata {
    name = "wealist-prod"
    labels = {
      "istio-injection" = "enabled"  # Sidecar 자동 주입
    }
  }
}
```

수동 생성이 필요한 경우:
```bash
# 네임스페이스 생성
kubectl create namespace wealist-prod

# Sidecar 자동 주입 활성화
kubectl label namespace wealist-prod istio-injection=enabled

# 확인
kubectl get namespace wealist-prod --show-labels
```

### 5. ArgoCD 설치 (선택)

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

## 출력 값

```hcl
cluster_name     = "wealist-prod-eks"
cluster_endpoint = "https://xxx.gr7.ap-northeast-2.eks.amazonaws.com"
oidc_provider_arn = "arn:aws:iam::xxx:oidc-provider/..."

# kubectl 설정 명령
configure_kubectl = "aws eks update-kubeconfig --name wealist-prod-eks --region ap-northeast-2"
```

## 비용

| 리소스 | 예상 비용/월 |
|--------|-------------|
| EKS Control Plane | $73 |
| Spot Nodes (3x t3.medium) | ~$30 |
| EBS (50GB x 3) | ~$15 |
| **합계** | ~$118 |

## State 위치

```
s3://wealist-tf-state-advanced-k8s/prod/compute/terraform.tfstate
```

## 트러블슈팅

### 클러스터 접근 불가

```bash
# kubeconfig 재설정
aws eks update-kubeconfig --name wealist-prod-eks --region ap-northeast-2

# IAM 확인
aws sts get-caller-identity

# 클러스터 상태 확인
aws eks describe-cluster --name wealist-prod-eks --query "cluster.status"
```

### 노드 Not Ready

```bash
# 노드 상태 확인
kubectl get nodes
kubectl describe node <node-name>

# VPC CNI 확인
kubectl get pods -n kube-system -l k8s-app=aws-node
kubectl logs -n kube-system -l k8s-app=aws-node
```

### Pod Identity 작동 안 함

```bash
# Pod Identity Agent 확인
kubectl get pods -n kube-system -l app.kubernetes.io/name=eks-pod-identity-agent

# Association 확인
aws eks list-pod-identity-associations --cluster-name wealist-prod-eks
```

### Add-on 업데이트 실패

```bash
# Add-on 상태 확인
aws eks describe-addon --cluster-name wealist-prod-eks --addon-name vpc-cni

# 강제 업데이트
aws eks update-addon \
  --cluster-name wealist-prod-eks \
  --addon-name vpc-cni \
  --resolve-conflicts OVERWRITE
```

### Istio Sidecar 문제

```bash
# istiod 상태 확인
kubectl get pods -n istio-system -l app=istiod

# Sidecar 주입 확인
kubectl get pods -n wealist-prod -o jsonpath='{.items[*].spec.containers[*].name}' | tr ' ' '\n' | grep -c istio-proxy

# Sidecar 로그 확인
kubectl logs deploy/<service-name> -n wealist-prod -c istio-proxy --tail=50

# Envoy 설정 확인
istioctl proxy-config all deploy/<service-name> -n wealist-prod
```

## 관련 문서

- Foundation 레이어: [../foundation/README.md](../foundation/README.md)
- Pod Identity 모듈: [../../modules/pod-identity/README.md](../../modules/pod-identity/README.md)
- Istio Sidecar 가이드: [../../../.claude/docs/ISTIO_SIDECAR.md](../../../.claude/docs/ISTIO_SIDECAR.md)
