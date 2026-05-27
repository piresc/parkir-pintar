module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"

  cluster_name    = "piresc-parkir"
  cluster_version = "1.32"

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  cluster_endpoint_public_access = true

  enable_cluster_creator_admin_permissions = true

  cluster_addons = {
    coredns = {
      most_recent = true
    }
    kube-proxy = {
      most_recent = true
    }
    vpc-cni = {
      most_recent = true
    }
  }

  fargate_profiles = {
    default = {
      name = "default"
      selectors = [
        {
          namespace = "pirescer-parkir-pintar"
          labels = {
            workload = "service"
          }
        }
      ]
    }
  }

  eks_managed_node_groups = {
    nats = {
      instance_types = ["t3.medium"]

      desired_size = var.nats_ec2_desired_size
      max_size     = var.nats_ec2_max_size
      min_size     = 1

      block_device_mappings = {
        xvda = {
          device_name = "/dev/xvda"
          ebs = {
            volume_type           = "gp3"
            volume_size           = 30
            delete_on_termination = true
          }
        }
      }

      labels = {
        role = "nats"
      }

      taints = {
        nats = {
          key    = "nats"
          value  = "true"
          effect = "NO_SCHEDULE"
        }
      }
    }
  }

  tags = {
    Project     = var.project_name
    Environment = var.environment
  }
}

resource "aws_iam_policy" "alb_controller" {
  name        = "${var.project_name}-${var.environment}-alb-controller"
  description = "IAM policy for AWS Load Balancer Controller"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["ec2:Describe*"]
        Resource = ["*"]
      },
      {
        Effect   = "Allow"
        Action   = ["elasticloadbalancing:*"]
        Resource = ["*"]
      },
      {
        Effect   = "Allow"
        Action   = ["acm:*"]
        Resource = ["*"]
      },
      {
        Effect   = "Allow"
        Action   = ["shield:*"]
        Resource = ["*"]
      },
      {
        Effect   = "Allow"
        Action   = ["wafv2:*"]
        Resource = ["*"]
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "alb_controller" {
  policy_arn = aws_iam_policy.alb_controller.arn
  role       = module.eks.cluster_iam_role_name
}

resource "kubernetes_namespace" "parkir_pintar" {
  depends_on = [module.eks]

  metadata {
    name = "pirescer-parkir-pintar"
    annotations = {
      "scheduler.alpha.kubernetes.io/node-selector" = "workload=service"
    }
  }
}


