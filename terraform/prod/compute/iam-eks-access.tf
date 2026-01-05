# =============================================================================
# EKS Cluster Access - IAM Groups, Roles, and Access Entries
# =============================================================================
# IAM Group 기반 접근 관리:
# - Terraform: Group, Role, Access Entry 생성
# - AWS Console: 팀원을 Group에 추가/제거
#
# 권한 레벨:
# - Admin: 전체 클러스터 관리 (AmazonEKSClusterAdminPolicy)
# - Developer: wealist-prod, argocd 네임스페이스 (AmazonEKSEditPolicy)
# - ReadOnly: 전체 조회 (AmazonEKSViewPolicy)

# =============================================================================
# IAM Groups (팀원을 Console에서 추가/제거)
# =============================================================================

resource "aws_iam_group" "eks_admin" {
  name = "${local.name_prefix}-eks-admin"
  path = "/eks-access/"
}

resource "aws_iam_group" "eks_developer" {
  name = "${local.name_prefix}-eks-developer"
  path = "/eks-access/"
}

resource "aws_iam_group" "eks_readonly" {
  name = "${local.name_prefix}-eks-readonly"
  path = "/eks-access/"
}

# =============================================================================
# IAM Roles (EKS Access Entry용)
# =============================================================================

# Admin Role - 전체 클러스터 관리 권한
resource "aws_iam_role" "eks_admin" {
  name = "${local.name_prefix}-eks-admin"

  # 같은 계정의 모든 IAM principal이 AssumeRole 가능
  # Group Policy에서 이 Role만 AssumeRole 허용하도록 제한
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
      }
      Action = "sts:AssumeRole"
      Condition = {
        StringEquals = {
          "aws:PrincipalType" = "User"
        }
      }
    }]
  })

  tags = local.common_tags
}

# Developer Role - 네임스페이스 제한 배포 권한
resource "aws_iam_role" "eks_developer" {
  name = "${local.name_prefix}-eks-developer"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
      }
      Action = "sts:AssumeRole"
      Condition = {
        StringEquals = {
          "aws:PrincipalType" = "User"
        }
      }
    }]
  })

  tags = local.common_tags
}

# ReadOnly Role - 조회 전용
resource "aws_iam_role" "eks_readonly" {
  name = "${local.name_prefix}-eks-readonly"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
      }
      Action = "sts:AssumeRole"
      Condition = {
        StringEquals = {
          "aws:PrincipalType" = "User"
        }
      }
    }]
  })

  tags = local.common_tags
}

# =============================================================================
# IAM Group Policies - AssumeRole 허용
# =============================================================================

resource "aws_iam_group_policy" "admin_assume_role" {
  name  = "assume-eks-admin-role"
  group = aws_iam_group.eks_admin.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "sts:AssumeRole"
      Resource = aws_iam_role.eks_admin.arn
    }]
  })
}

resource "aws_iam_group_policy" "developer_assume_role" {
  name  = "assume-eks-developer-role"
  group = aws_iam_group.eks_developer.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "sts:AssumeRole"
      Resource = aws_iam_role.eks_developer.arn
    }]
  })
}

resource "aws_iam_group_policy" "readonly_assume_role" {
  name  = "assume-eks-readonly-role"
  group = aws_iam_group.eks_readonly.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "sts:AssumeRole"
      Resource = aws_iam_role.eks_readonly.arn
    }]
  })
}

# =============================================================================
# EKS Access Entries (Role → Cluster 접근 권한)
# =============================================================================

# Admin Access Entry - 전체 클러스터 관리
resource "aws_eks_access_entry" "admin" {
  cluster_name  = module.eks.cluster_name
  principal_arn = aws_iam_role.eks_admin.arn
  type          = "STANDARD"

  tags = local.common_tags
}

resource "aws_eks_access_policy_association" "admin" {
  cluster_name  = module.eks.cluster_name
  principal_arn = aws_iam_role.eks_admin.arn
  policy_arn    = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy"

  access_scope {
    type = "cluster"
  }

  depends_on = [aws_eks_access_entry.admin]
}

# Developer Access Entry - 네임스페이스 제한 배포
resource "aws_eks_access_entry" "developer" {
  cluster_name  = module.eks.cluster_name
  principal_arn = aws_iam_role.eks_developer.arn
  type          = "STANDARD"

  tags = local.common_tags
}

resource "aws_eks_access_policy_association" "developer" {
  cluster_name  = module.eks.cluster_name
  principal_arn = aws_iam_role.eks_developer.arn
  policy_arn    = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSEditPolicy"

  access_scope {
    type       = "namespace"
    namespaces = ["wealist-prod", "argocd"]
  }

  depends_on = [aws_eks_access_entry.developer]
}

# ReadOnly Access Entry - 전체 조회
resource "aws_eks_access_entry" "readonly" {
  cluster_name  = module.eks.cluster_name
  principal_arn = aws_iam_role.eks_readonly.arn
  type          = "STANDARD"

  tags = local.common_tags
}

resource "aws_eks_access_policy_association" "readonly" {
  cluster_name  = module.eks.cluster_name
  principal_arn = aws_iam_role.eks_readonly.arn
  policy_arn    = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSViewPolicy"

  access_scope {
    type = "cluster"
  }

  depends_on = [aws_eks_access_entry.readonly]
}
