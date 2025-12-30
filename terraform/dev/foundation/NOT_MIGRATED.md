# 아직 마이그레이션되지 않음

이 디렉토리는 새로운 구조로 준비되었지만, 아직 사용되지 않습니다.

## 현재 사용 중인 디렉토리

```
terraform/dev-environment/
```

State 경로: `s3://wealist-tf-state-advanced-k8s/dev-enviorment/terraform.tfstate`

## 마이그레이션 방법

마이그레이션하려면:

```bash
# 1. State 복사
aws s3 cp s3://wealist-tf-state-advanced-k8s/dev-enviorment/terraform.tfstate \
          s3://wealist-tf-state-advanced-k8s/dev/foundation/terraform.tfstate

# 2. 새 디렉토리에서 init
cd terraform/dev/foundation
terraform init

# 3. 상태 확인
terraform plan  # 변경사항 없어야 함

# 4. 기존 디렉토리 삭제
rm -rf terraform/dev-environment/
```
