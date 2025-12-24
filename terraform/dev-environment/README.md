cd terraform/dev-environment

# terraform.tfvars 먼저 생성

cp terraform.tfvars.example terraform.tfvars

# 필요시 실제 값으로 수정

# 초기화

terraform init

# 시크릿 모듈만 생성(\*)

terraform apply -target=module.secrets
