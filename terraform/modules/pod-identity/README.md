# Pod Identity Module

EKS Pod Identity를 사용하여 Kubernetes Pod에 AWS IAM 권한을 부여하는 모듈입니다.

## 개요

Pod Identity는 IRSA(IAM Roles for Service Accounts)의 후속 기술로, 2024년부터 AWS에서 권장하는 방식입니다.

### Pod Identity vs IRSA

| 특성 | Pod Identity | IRSA |
|------|-------------|------|
| 설정 복잡도 | 낮음 | 높음 (OIDC 설정 필요) |
| 크로스 계정 | 지원 | 지원 |
| 토큰 갱신 | 자동 (60분) | 자동 (12시간) |
| EKS 버전 | 1.24+ | 1.14+ |
| 권장 사항 | **신규 클러스터 권장** | 기존 클러스터 호환 |

## 사용법

### 기본 사용

```hcl
module "pod_identity_example" {
  source = "../../modules/pod-identity"

  name            = "my-app"
  cluster_name    = "wealist-prod-eks"
  namespace       = "my-namespace"
  service_account = "my-service-account"

  inline_policies = {
    s3-access = jsonencode({
      Version = "2012-10-17"
      Statement = [{
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:PutObject"]
        Resource = "arn:aws:s3:::my-bucket/*"
      }]
    })
  }

  tags = {
    Environment = "prod"
  }
}
```

### AWS 관리형 정책 사용

```hcl
module "pod_identity_with_managed_policy" {
  source = "../../modules/pod-identity"

  name            = "alb-controller"
  cluster_name    = "wealist-prod-eks"
  namespace       = "kube-system"
  service_account = "aws-load-balancer-controller"

  policy_arns = [
    "arn:aws:iam::aws:policy/ElasticLoadBalancingFullAccess"
  ]

  tags = {
    Environment = "prod"
  }
}
```

### 관리형 + 인라인 정책 조합

```hcl
module "pod_identity_combined" {
  source = "../../modules/pod-identity"

  name            = "external-secrets"
  cluster_name    = "wealist-prod-eks"
  namespace       = "external-secrets"
  service_account = "external-secrets"

  policy_arns = [
    "arn:aws:iam::aws:policy/SecretsManagerReadWrite"
  ]

  inline_policies = {
    kms-access = jsonencode({
      Version = "2012-10-17"
      Statement = [{
        Effect   = "Allow"
        Action   = ["kms:Decrypt", "kms:DescribeKey"]
        Resource = "arn:aws:kms:ap-northeast-2:123456789:key/xxx"
      }]
    })
  }

  tags = {
    Environment = "prod"
  }
}
```

## 입력 변수

| 이름 | 타입 | 필수 | 기본값 | 설명 |
|------|------|------|--------|------|
| `name` | string | Yes | - | 리소스 이름 prefix |
| `cluster_name` | string | Yes | - | EKS 클러스터 이름 |
| `namespace` | string | Yes | - | Kubernetes 네임스페이스 |
| `service_account` | string | Yes | - | ServiceAccount 이름 |
| `policy_arns` | list(string) | No | [] | 연결할 관리형 정책 ARN 목록 |
| `inline_policies` | map(string) | No | {} | 인라인 정책 (이름 → JSON 맵) |
| `tags` | map(string) | No | {} | 리소스 태그 |

## 출력 값

| 이름 | 설명 |
|------|------|
| `role_arn` | 생성된 IAM 역할 ARN |
| `role_name` | 생성된 IAM 역할 이름 |
| `association_id` | EKS Pod Identity Association ID |

## 전제 조건

### EKS Add-on 설치 필요

Pod Identity를 사용하려면 `eks-pod-identity-agent` Add-on이 설치되어 있어야 합니다:

```hcl
cluster_addons = {
  eks-pod-identity-agent = {
    addon_version = "v1.3.2-eksbuild.2"
  }
}
```

### ServiceAccount 생성

Pod Identity는 기존 ServiceAccount와 연결됩니다. ServiceAccount가 미리 생성되어 있어야 합니다:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-service-account
  namespace: my-namespace
```

## weAlist 프로젝트 Pod Identity 목록

| 서비스 | Namespace | ServiceAccount | 권한 |
|--------|-----------|----------------|------|
| ALB Controller | kube-system | aws-load-balancer-controller | ELB, EC2, ACM, WAF |
| External Secrets | external-secrets | external-secrets | Secrets Manager, SSM, KMS |
| External DNS | external-dns | external-dns | Route53 |
| cert-manager | cert-manager | cert-manager | Route53 (DNS-01) |
| storage-service | wealist-prod | storage-service | S3 |
| Cluster Autoscaler | kube-system | cluster-autoscaler | ASG, EC2 |

## 디버깅

### Pod Identity 연결 확인

```bash
# Pod Identity 연결 목록
aws eks list-pod-identity-associations --cluster-name wealist-prod-eks

# 특정 연결 상세
aws eks describe-pod-identity-association \
  --cluster-name wealist-prod-eks \
  --association-id <association-id>
```

### Pod에서 권한 확인

```bash
# Pod 내부에서 IAM 자격 증명 확인
kubectl exec -it <pod-name> -n <namespace> -- aws sts get-caller-identity
```

### 일반적인 문제

1. **Pod Identity Agent 미설치**
   ```bash
   kubectl get pods -n kube-system -l app.kubernetes.io/name=eks-pod-identity-agent
   ```

2. **ServiceAccount 불일치**
   - Pod Identity Association의 ServiceAccount와 Pod의 ServiceAccount가 일치해야 함

3. **IAM 권한 부족**
   - inline_policies 또는 policy_arns 확인
