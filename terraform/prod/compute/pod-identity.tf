# =============================================================================
# Pod Identity Associations
# =============================================================================
# EKS Pod Identity를 사용하여 각 서비스에 AWS 권한 부여
# IRSA 대신 Pod Identity 사용 (2024-2025 권장 방식)

# -----------------------------------------------------------------------------
# AWS Load Balancer Controller
# -----------------------------------------------------------------------------
module "pod_identity_alb_controller" {
  source = "../../modules/pod-identity"

  name            = "${local.name_prefix}-alb-controller"
  cluster_name    = module.eks.cluster_name
  namespace       = "kube-system"
  service_account = "aws-load-balancer-controller"

  policy_arns = [
    "arn:aws:iam::aws:policy/ElasticLoadBalancingFullAccess"
  ]

  inline_policies = {
    alb-controller = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Sid    = "ALBControllerEC2Describe"
          Effect = "Allow"
          Action = [
            "ec2:DescribeAvailabilityZones",
            "ec2:DescribeSecurityGroups",
            "ec2:DescribeSubnets",
            "ec2:DescribeVpcs",
            "ec2:DescribeAccountAttributes",
            "ec2:DescribeInternetGateways",
            "ec2:DescribeTags",
            "ec2:DescribeInstances",
            "ec2:DescribeNetworkInterfaces",
            "ec2:DescribeCoipPools",
            "ec2:GetCoipPoolUsage"
          ]
          Resource = "*"
        },
        {
          Sid    = "ALBControllerEC2SecurityGroup"
          Effect = "Allow"
          Action = [
            "ec2:CreateSecurityGroup",
            "ec2:DeleteSecurityGroup",
            "ec2:AuthorizeSecurityGroupIngress",
            "ec2:RevokeSecurityGroupIngress",
            "ec2:CreateTags"
          ]
          Resource = "*"
        },
        {
          Sid    = "ALBControllerIAM"
          Effect = "Allow"
          Action = [
            "iam:CreateServiceLinkedRole"
          ]
          Resource = "*"
          Condition = {
            StringEquals = {
              "iam:AWSServiceName" = "elasticloadbalancing.amazonaws.com"
            }
          }
        },
        {
          Sid    = "ALBControllerACM"
          Effect = "Allow"
          Action = [
            "acm:ListCertificates",
            "acm:DescribeCertificate"
          ]
          Resource = "*"
        },
        {
          Sid    = "ALBControllerCognito"
          Effect = "Allow"
          Action = [
            "cognito-idp:DescribeUserPoolClient"
          ]
          Resource = "*"
        },
        {
          Sid    = "ALBControllerWAF"
          Effect = "Allow"
          Action = [
            "waf-regional:*",
            "wafv2:*"
          ]
          Resource = "*"
        },
        {
          Sid    = "ALBControllerShield"
          Effect = "Allow"
          Action = [
            "shield:*"
          ]
          Resource = "*"
        }
      ]
    })
  }

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# External Secrets Operator
# -----------------------------------------------------------------------------
module "pod_identity_external_secrets" {
  source = "../../modules/pod-identity"

  name            = "${local.name_prefix}-external-secrets"
  cluster_name    = module.eks.cluster_name
  namespace       = "external-secrets"
  service_account = "external-secrets"

  inline_policies = {
    secrets-access = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Sid    = "SecretsManagerAccess"
          Effect = "Allow"
          Action = [
            "secretsmanager:GetSecretValue",
            "secretsmanager:DescribeSecret",
            "secretsmanager:ListSecrets"
          ]
          Resource = [
            "arn:aws:secretsmanager:${var.aws_region}:${data.aws_caller_identity.current.account_id}:secret:wealist/*",
            "arn:aws:secretsmanager:${var.aws_region}:${data.aws_caller_identity.current.account_id}:secret:rds!*"
          ]
        },
        {
          Sid    = "SSMParameterAccess"
          Effect = "Allow"
          Action = [
            "ssm:GetParameter",
            "ssm:GetParameters",
            "ssm:GetParametersByPath"
          ]
          Resource = "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter/wealist/*"
        },
        {
          Sid    = "KMSDecrypt"
          Effect = "Allow"
          Action = [
            "kms:Decrypt",
            "kms:DescribeKey"
          ]
          Resource = local.kms_key_arn
        }
      ]
    })
  }

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# External DNS
# -----------------------------------------------------------------------------
module "pod_identity_external_dns" {
  source = "../../modules/pod-identity"

  name            = "${local.name_prefix}-external-dns"
  cluster_name    = module.eks.cluster_name
  namespace       = "external-dns"
  service_account = "external-dns"

  inline_policies = {
    route53-access = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Sid    = "Route53RecordChanges"
          Effect = "Allow"
          Action = [
            "route53:ChangeResourceRecordSets"
          ]
          Resource = "arn:aws:route53:::hostedzone/*"
        },
        {
          Sid    = "Route53ListZones"
          Effect = "Allow"
          Action = [
            "route53:ListHostedZones",
            "route53:ListResourceRecordSets",
            "route53:ListTagsForResource"
          ]
          Resource = "*"
        }
      ]
    })
  }

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# cert-manager
# -----------------------------------------------------------------------------
module "pod_identity_cert_manager" {
  source = "../../modules/pod-identity"

  name            = "${local.name_prefix}-cert-manager"
  cluster_name    = module.eks.cluster_name
  namespace       = "cert-manager"
  service_account = "cert-manager"

  inline_policies = {
    route53-dns01 = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Sid    = "Route53DNS01Challenge"
          Effect = "Allow"
          Action = [
            "route53:GetChange"
          ]
          Resource = "arn:aws:route53:::change/*"
        },
        {
          Sid    = "Route53RecordChanges"
          Effect = "Allow"
          Action = [
            "route53:ChangeResourceRecordSets",
            "route53:ListResourceRecordSets"
          ]
          Resource = "arn:aws:route53:::hostedzone/*"
        },
        {
          Sid    = "Route53ListZones"
          Effect = "Allow"
          Action = [
            "route53:ListHostedZones"
          ]
          Resource = "*"
        }
      ]
    })
  }

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# storage-service (S3 접근)
# -----------------------------------------------------------------------------
module "pod_identity_storage_service" {
  source = "../../modules/pod-identity"

  name            = "${local.name_prefix}-storage-service"
  cluster_name    = module.eks.cluster_name
  namespace       = "wealist-prod"
  service_account = "storage-service"

  inline_policies = {
    s3-access = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Sid    = "S3BucketAccess"
          Effect = "Allow"
          Action = [
            "s3:GetObject",
            "s3:PutObject",
            "s3:DeleteObject",
            "s3:ListBucket"
          ]
          Resource = [
            data.terraform_remote_state.foundation.outputs.s3_bucket_arn,
            "${data.terraform_remote_state.foundation.outputs.s3_bucket_arn}/*"
          ]
        },
        {
          Sid    = "KMSAccess"
          Effect = "Allow"
          Action = [
            "kms:GenerateDataKey",
            "kms:Decrypt",
            "kms:DescribeKey"
          ]
          Resource = local.kms_key_arn
        }
      ]
    })
  }

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# board-service (S3 Presigned URL 생성 - 첨부파일)
# -----------------------------------------------------------------------------
module "pod_identity_board_service" {
  source = "../../modules/pod-identity"

  name            = "${local.name_prefix}-board-service"
  cluster_name    = module.eks.cluster_name
  namespace       = "wealist-prod"
  service_account = "board-service"

  inline_policies = {
    s3-access = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Sid    = "S3BucketAccess"
          Effect = "Allow"
          Action = [
            "s3:GetObject",
            "s3:PutObject",
            "s3:DeleteObject",
            "s3:ListBucket"
          ]
          Resource = [
            data.terraform_remote_state.foundation.outputs.s3_bucket_arn,
            "${data.terraform_remote_state.foundation.outputs.s3_bucket_arn}/*"
          ]
        },
        {
          Sid    = "KMSAccess"
          Effect = "Allow"
          Action = [
            "kms:GenerateDataKey",
            "kms:Decrypt",
            "kms:DescribeKey"
          ]
          Resource = local.kms_key_arn
        }
      ]
    })
  }

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# user-service (S3 Presigned URL 생성 - 프로필 이미지)
# -----------------------------------------------------------------------------
module "pod_identity_user_service" {
  source = "../../modules/pod-identity"

  name            = "${local.name_prefix}-user-service"
  cluster_name    = module.eks.cluster_name
  namespace       = "wealist-prod"
  service_account = "user-service"

  inline_policies = {
    s3-access = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Sid    = "S3BucketAccess"
          Effect = "Allow"
          Action = [
            "s3:GetObject",
            "s3:PutObject",
            "s3:DeleteObject",
            "s3:ListBucket"
          ]
          Resource = [
            data.terraform_remote_state.foundation.outputs.s3_bucket_arn,
            "${data.terraform_remote_state.foundation.outputs.s3_bucket_arn}/*"
          ]
        },
        {
          Sid    = "KMSAccess"
          Effect = "Allow"
          Action = [
            "kms:GenerateDataKey",
            "kms:Decrypt",
            "kms:DescribeKey"
          ]
          Resource = local.kms_key_arn
        }
      ]
    })
  }

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# chat-service (S3 접근 - 채팅 첨부파일)
# -----------------------------------------------------------------------------
module "pod_identity_chat_service" {
  source = "../../modules/pod-identity"

  name            = "${local.name_prefix}-chat-service"
  cluster_name    = module.eks.cluster_name
  namespace       = "wealist-prod"
  service_account = "chat-service"

  inline_policies = {
    s3-access = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Sid    = "S3BucketAccess"
          Effect = "Allow"
          Action = [
            "s3:GetObject",
            "s3:PutObject",
            "s3:DeleteObject",
            "s3:ListBucket"
          ]
          Resource = [
            data.terraform_remote_state.foundation.outputs.s3_bucket_arn,
            "${data.terraform_remote_state.foundation.outputs.s3_bucket_arn}/*"
          ]
        },
        {
          Sid    = "KMSAccess"
          Effect = "Allow"
          Action = [
            "kms:GenerateDataKey",
            "kms:Decrypt",
            "kms:DescribeKey"
          ]
          Resource = local.kms_key_arn
        }
      ]
    })
  }

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# video-service (S3 접근 - 녹화 파일 저장)
# -----------------------------------------------------------------------------
module "pod_identity_video_service" {
  source = "../../modules/pod-identity"

  name            = "${local.name_prefix}-video-service"
  cluster_name    = module.eks.cluster_name
  namespace       = "wealist-prod"
  service_account = "video-service"

  inline_policies = {
    s3-access = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Sid    = "S3BucketAccess"
          Effect = "Allow"
          Action = [
            "s3:GetObject",
            "s3:PutObject",
            "s3:DeleteObject",
            "s3:ListBucket"
          ]
          Resource = [
            data.terraform_remote_state.foundation.outputs.s3_bucket_arn,
            "${data.terraform_remote_state.foundation.outputs.s3_bucket_arn}/*"
          ]
        },
        {
          Sid    = "KMSAccess"
          Effect = "Allow"
          Action = [
            "kms:GenerateDataKey",
            "kms:Decrypt",
            "kms:DescribeKey"
          ]
          Resource = local.kms_key_arn
        }
      ]
    })
  }

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# Cluster Autoscaler
# -----------------------------------------------------------------------------
module "pod_identity_cluster_autoscaler" {
  source = "../../modules/pod-identity"

  name            = "${local.name_prefix}-cluster-autoscaler"
  cluster_name    = module.eks.cluster_name
  namespace       = "kube-system"
  service_account = "cluster-autoscaler"

  inline_policies = {
    autoscaler = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Sid    = "AutoscalerDescribe"
          Effect = "Allow"
          Action = [
            "autoscaling:DescribeAutoScalingGroups",
            "autoscaling:DescribeAutoScalingInstances",
            "autoscaling:DescribeLaunchConfigurations",
            "autoscaling:DescribeScalingActivities",
            "autoscaling:DescribeTags",
            "ec2:DescribeImages",
            "ec2:DescribeInstanceTypes",
            "ec2:DescribeLaunchTemplateVersions",
            "ec2:GetInstanceTypesFromInstanceRequirements",
            "eks:DescribeNodegroup"
          ]
          Resource = "*"
        },
        {
          Sid    = "AutoscalerScale"
          Effect = "Allow"
          Action = [
            "autoscaling:SetDesiredCapacity",
            "autoscaling:TerminateInstanceInAutoScalingGroup"
          ]
          Resource = "*"
          Condition = {
            StringEquals = {
              "aws:ResourceTag/k8s.io/cluster-autoscaler/${local.cluster_name}" = "owned"
            }
          }
        }
      ]
    })
  }

  tags = local.common_tags
}
