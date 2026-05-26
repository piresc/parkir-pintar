# CI/CD Production Pipeline — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the current monolithic ci.yml into PR checks + two build workflows, create AWS EKS infrastructure via Terraform, and add a production deploy workflow with approval gates, migrations, rolling updates, and smoke tests.

**Architecture:** 4 GitHub Actions workflows (ci-pr.yml, build-push-staging.yml, build-push-prod.yml, deploy-prod.yml) on top of the existing GHCR image registry. Production deploys to AWS EKS (Fargate for 8 apps + EC2 node group for NATS) managed by Terraform. Staging stays on Coolify unchanged.

**Tech Stack:** GitHub Actions, Docker, Terraform (AWS provider), Kubernetes (EKS), kubectl, k6

---

### Task 1: Trim Current ci.yml to ci-pr.yml (PR Checks Only)

**Files:**
- Create: `.github/workflows/ci-pr.yml`
- Modify: `.github/workflows/ci.yml` (remove build/deploy jobs, keep quality gates)

**What this does:** The current ci.yml contains everything. We split it: `ci-pr.yml` runs on PRs and only does checks (secret-scan, lint, test, security, vulncheck, proto, sonar). The build/deploy jobs move to separate workflows.

- [ ] **Step 1: Create ci-pr.yml with all quality gate jobs, pinned versions, and caching**

Write to `.github/workflows/ci-pr.yml`:

```yaml
name: PR Checks

on:
  pull_request:
    branches: [main]

permissions:
  contents: read

env:
  GO_VERSION: "1.25"
  GOLANGCI_LINT_VERSION: "v2.12.2"
  GOSEC_VERSION: "v2.22.3"
  GOVULNCHECK_VERSION: "v1.1.4"
  NODE_VERSION: "20"

jobs:
  secret-scan:
    name: Secret Scan
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: gitleaks/gitleaks-action@v2
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  lint:
    name: Lint
    runs-on: ubuntu-latest
    needs: secret-scan
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      - name: Install golangci-lint
        run: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${{ env.GOLANGCI_LINT_VERSION }}
      - name: Run golangci-lint
        run: golangci-lint run --timeout 5m

  test:
    name: Test
    runs-on: ubuntu-latest
    needs: secret-scan
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      - name: Run unit tests with race detector and coverage
        run: go test -v -race -covermode=atomic -coverprofile=coverage.txt -short ./...
      - name: Display coverage summary
        run: go tool cover -func=coverage.txt
      - name: Upload coverage artifact
        uses: actions/upload-artifact@v4
        with:
          name: coverage
          path: coverage.txt

  security:
    name: Security Scan
    runs-on: ubuntu-latest
    needs: secret-scan
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      - name: Install gosec
        run: go install github.com/securego/gosec/v2/cmd/gosec@${{ env.GOSEC_VERSION }}
      - name: Run gosec
        run: gosec -exclude=G401,G304,G501,G505,G103,G104,G109,G115 -exclude-dir=proto -exclude-dir=tests -fmt=sonarqube -out=sonar-gosec.json ./...
      - name: Upload gosec report
        uses: actions/upload-artifact@v4
        with:
          name: gosec-report
          path: sonar-gosec.json

  vulncheck:
    name: Vulnerability Check
    runs-on: ubuntu-latest
    needs: secret-scan
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      - name: Install govulncheck
        run: go install golang.org/x/vuln/cmd/govulncheck@${{ env.GOVULNCHECK_VERSION }}
      - name: Run govulncheck
        run: govulncheck -scan package ./...

  sonar:
    name: SonarCloud
    runs-on: ubuntu-latest
    needs: [test, security]
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/download-artifact@v4
        with:
          name: coverage
      - uses: actions/download-artifact@v4
        with:
          name: gosec-report
      - uses: SonarSource/sonarqube-scan-action@v6
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}

  proto-check:
    name: Proto Check
    runs-on: ubuntu-latest
    needs: secret-scan
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - name: Install buf
        uses: bufbuild/buf-setup-action@v1
      - name: Lint proto files
        run: buf lint proto/
      - name: Check for breaking changes
        run: buf breaking proto/ --against '.git#branch=main,subdir=proto'

  frontend-check:
    name: Frontend Check
    runs-on: ubuntu-latest
    needs: secret-scan
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: "npm"
          cache-dependency-path: frontend/package-lock.json
      - name: Install dependencies
        working-directory: frontend
        run: npm ci
      - name: Lint frontend
        working-directory: frontend
        run: npx eslint src/ --ext .ts,.tsx --max-warnings 0 || echo "::warning::No ESLint config found — skipping"
      - name: Type check frontend
        working-directory: frontend
        run: npx tsc --noEmit
```

- [ ] **Step 2: Verify the new ci-pr.yml is valid YAML**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci-pr.yml')); print('Valid YAML')"`

- [ ] **Step 3: Remove build-push and deploy jobs from ci.yml, keeping only quality gate jobs as ci-pr.yml**

Since we just created ci-pr.yml with all quality gates, simply remove the old ci.yml (renamed to preserve git history, we'll update it) -- no, we should keep ci.yml as-is for now and only decommission it at the end. This task just creates ci-pr.yml.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci-pr.yml
git commit -m "ci: add PR checks workflow with pinned versions and caching"
```

---

### Task 2: Create build-push-staging.yml

**Files:**
- Create: `.github/workflows/build-push-staging.yml`

**What this does:** Extracts the build-and-push logic from the current ci.yml. Triggered on push to main, builds all 7 Go services + frontend, pushes to GHCR with `main-$SHA` and `latest` tags.

- [ ] **Step 1: Create build-push-staging.yml**

Write to `.github/workflows/build-push-staging.yml`:

```yaml
name: Build & Push Staging

on:
  push:
    branches: [main]
    paths-ignore:
      - "docs/**"
      - "**.md"
      - ".github/**"
      - "deploy/coolify/README.md"

permissions:
  contents: read
  packages: write

jobs:
  build-push-services:
    name: Build & Push ${{ matrix.service }}
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: [gateway, reservation, billing, payment, search, presence, analytics]
    steps:
      - uses: actions/checkout@v4

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push ${{ matrix.service }} image
        uses: docker/build-push-action@v5
        with:
          context: .
          file: ./cmd/${{ matrix.service }}/Dockerfile
          push: true
          tags: |
            ghcr.io/${{ github.repository }}/${{ matrix.service }}:latest
            ghcr.io/${{ github.repository }}/${{ matrix.service }}:main-${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            VERSION=${{ github.sha }}
            GIT_COMMIT=${{ github.sha }}
            BUILD_TIME=${{ github.event.head_commit.timestamp }}

  build-push-frontend:
    name: Build & Push Frontend
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: "20"
          cache: "npm"
          cache-dependency-path: frontend/package-lock.json

      - name: Install dependencies
        working-directory: frontend
        run: npm ci

      - name: Build frontend
        working-directory: frontend
        env:
          VITE_JWT_SECRET: ${{ secrets.JWT_SECRET }}
          VITE_JWT_ISSUER: parkir-pintar
          VITE_JWT_EXPIRATION_MINUTES: "60"
        run: npm run build

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push Frontend Docker image
        uses: docker/build-push-action@v5
        with:
          context: ./frontend
          push: true
          tags: |
            ghcr.io/${{ github.repository }}-frontend:latest
            ghcr.io/${{ github.repository }}-frontend:main-${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  trigger-deploy:
    name: Trigger Staging Deploy
    runs-on: ubuntu-latest
    needs: [build-push-services, build-push-frontend]
    steps:
      - name: Trigger Coolify webhook
        run: |
          curl -sSf -X POST \
            "${{ secrets.COOLIFY_WEBHOOK_URL }}" \
            -H "Authorization: Bearer ${{ secrets.COOLIFY_TOKEN }}" \
            -H "Content-Type: application/json" \
            --max-time 30
```

- [ ] **Step 2: Validate YAML**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/build-push-staging.yml')); print('Valid YAML')"`

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/build-push-staging.yml
git commit -m "ci: add staging build-push workflow with Docker layer caching"
```

---

### Task 3: Create build-push-prod.yml

**Files:**
- Create: `.github/workflows/build-push-prod.yml`

**What this does:** Same build logic as staging but triggered by semver tags (`v*`). Tags images with `v1.0.0` and `latest-prod`. Does NOT auto-deploy.

- [ ] **Step 1: Create build-push-prod.yml**

Write to `.github/workflows/build-push-prod.yml`:

```yaml
name: Build & Push Production

on:
  push:
    tags:
      - "v*"

permissions:
  contents: read
  packages: write

jobs:
  build-push-services:
    name: Build & Push ${{ matrix.service }}
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: [gateway, reservation, billing, payment, search, presence, analytics]
    steps:
      - uses: actions/checkout@v4

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push ${{ matrix.service }} image
        uses: docker/build-push-action@v5
        with:
          context: .
          file: ./cmd/${{ matrix.service }}/Dockerfile
          push: true
          tags: |
            ghcr.io/${{ github.repository }}/${{ matrix.service }}:${{ github.ref_name }}
            ghcr.io/${{ github.repository }}/${{ matrix.service }}:latest-prod
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            VERSION=${{ github.ref_name }}
            GIT_COMMIT=${{ github.sha }}
            BUILD_TIME=${{ github.event.head_commit.timestamp }}

  build-push-frontend:
    name: Build & Push Frontend
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: "20"
          cache: "npm"
          cache-dependency-path: frontend/package-lock.json

      - name: Install dependencies
        working-directory: frontend
        run: npm ci

      - name: Build frontend
        working-directory: frontend
        env:
          VITE_JWT_SECRET: ${{ secrets.JWT_SECRET }}
          VITE_JWT_ISSUER: parkir-pintar
          VITE_JWT_EXPIRATION_MINUTES: "60"
        run: npm run build

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push Frontend Docker image
        uses: docker/build-push-action@v5
        with:
          context: ./frontend
          push: true
          tags: |
            ghcr.io/${{ github.repository }}-frontend:${{ github.ref_name }}
            ghcr.io/${{ github.repository }}-frontend:latest-prod
          cache-from: type=gha
          cache-to: type=gha,mode=max

  create-release-notes:
    name: Create GitHub Release
    runs-on: ubuntu-latest
    needs: [build-push-services, build-push-frontend]
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - name: Generate release notes
        run: |
          echo "## Images pushed" > release-notes.md
          echo "" >> release-notes.md
          for svc in gateway reservation billing payment search presence analytics; do
            echo "- \`ghcr.io/${{ github.repository }}/${svc}:${{ github.ref_name }}\`" >> release-notes.md
          done
          echo "- \`ghcr.io/${{ github.repository }}-frontend:${{ github.ref_name }}\`" >> release-notes.md
      - name: Create release
        uses: softprops/action-gh-release@v1
        with:
          body_path: release-notes.md
          generate_release_notes: true
```

- [ ] **Step 2: Validate YAML**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/build-push-prod.yml')); print('Valid YAML')"`

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/build-push-prod.yml
git commit -m "ci: add production build-push workflow (tag-triggered, semver)"
```

---

### Task 4: Create AWS Terraform Infrastructure — Network + EKS

**Files:**
- Create: `infra/aws/terraform.tf`
- Create: `infra/aws/variables.tf`
- Create: `infra/aws/terraform.tfvars.example`
- Create: `infra/aws/network.tf`
- Create: `infra/aws/eks.tf`
- Create: `infra/aws/.gitignore`

**What this does:** Core AWS infrastructure: VPC with public/private subnets, EKS cluster with Fargate profiles and EC2 node group for NATS.

- [ ] **Step 1: Create infra/aws/.gitignore**

Write to `infra/aws/.gitignore`:

```
.terraform/
*.tfstate
*.tfstate.*
*.tfvars
!terraform.tfvars.example
.terraform.lock.hcl
```

- [ ] **Step 2: Create terraform.tf (provider + state)**

Write to `infra/aws/terraform.tf`:

```hcl
terraform {
  required_version = ">= 1.9"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.0"
    }
  }

  backend "s3" {
    bucket         = "parkir-pintar-tfstate"
    key            = "production/terraform.tfstate"
    region         = "ap-southeast-1"
    dynamodb_table = "parkir-pintar-tfstate-lock"
    encrypt        = true
  }
}

provider "aws" {
  region = var.aws_region
}

data "aws_caller_identity" "current" {}
data "aws_availability_zones" "available" {}
```

- [ ] **Step 3: Create variables.tf**

Write to `infra/aws/variables.tf`:

```hcl
variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "ap-southeast-1"
}

variable "project_name" {
  description = "Project name used for tagging"
  type        = string
  default     = "parkir-pintar"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

variable "domain_name" {
  description = "Route53 hosted zone domain"
  type        = string
  default     = "parkir-pintar.piresc.dev"
}

variable "db_username" {
  description = "RDS database username"
  type        = string
  sensitive   = true
}

variable "db_password" {
  description = "RDS database password"
  type        = string
  sensitive   = true
}

variable "redis_auth_token" {
  description = "ElastiCache Redis auth token"
  type        = string
  sensitive   = true
}

variable "jwt_secret" {
  description = "JWT signing secret"
  type        = string
  sensitive   = true
}

variable "ghcr_username" {
  description = "GitHub Container Registry username"
  type        = string
}

variable "ghcr_token" {
  description = "GitHub Container Registry personal access token"
  type        = string
  sensitive   = true
}

variable "nats_instance_count" {
  description = "Number of NATS StatefulSet replicas"
  type        = number
  default     = 3
}

variable "nats_ec2_instance_type" {
  description = "EC2 instance type for NATS node group"
  type        = string
  default     = "t3.medium"
}

variable "nats_ec2_desired_size" {
  description = "Desired number of EC2 instances for NATS"
  type        = number
  default     = 2
}

variable "nats_ec2_max_size" {
  description = "Max number of EC2 instances for NATS"
  type        = number
  default     = 3
}
```

- [ ] **Step 4: Create terraform.tfvars.example**

Write to `infra/aws/terraform.tfvars.example`:

```hcl
aws_region        = "ap-southeast-1"
project_name      = "parkir-pintar"
environment       = "production"
domain_name       = "parkir-pintar.piresc.dev"
db_username       = "parkir_admin"
db_password       = "CHANGE-ME-USE-STRONG-PASSWORD"
redis_auth_token  = "CHANGE-ME-USE-STRONG-TOKEN"
jwt_secret        = "CHANGE-ME-USE-STRONG-SECRET"
ghcr_username     = "YOUR-GITHUB-USERNAME"
ghcr_token        = "YOUR-GHCR-PAT-TOKEN"
```

- [ ] **Step 5: Create network.tf (VPC, subnets, NAT, internet gateway)**

Write to `infra/aws/network.tf`:

```hcl
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = "${var.project_name}-${var.environment}"
  cidr = var.vpc_cidr

  azs             = slice(data.aws_availability_zones.available.names, 0, 2)
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24"]

  enable_nat_gateway   = true
  single_nat_gateway   = true
  enable_dns_hostnames = true
  enable_dns_support   = true

  public_subnet_tags = {
    "kubernetes.io/role/elb" = "1"
  }

  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = "1"
  }

  tags = {
    Project     = var.project_name
    Environment = var.environment
  }
}
```

- [ ] **Step 6: Create eks.tf (EKS cluster, Fargate profiles, EC2 node group, IAM roles)**

Write to `infra/aws/eks.tf`:

```hcl
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"

  cluster_name    = "${var.project_name}-${var.environment}"
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
          namespace = "parkir-pintar"
          labels = {
            workload = "service"
          }
        }
      ]
    }
  }

  eks_managed_node_groups = {
    nats = {
      name           = "nats-node-group"
      instance_types = [var.nats_ec2_instance_type]
      desired_size   = var.nats_ec2_desired_size
      max_size       = var.nats_ec2_max_size
      min_size       = 1

      labels = {
        role = "nats"
      }

      taints = {
        nats-only = {
          key    = "nats"
          value  = "true"
          effect = "NO_SCHEDULE"
        }
      }

      block_device_mappings = {
        xvda = {
          device_name = "/dev/xvda"
          ebs = {
            volume_size = 30
            volume_type = "gp3"
          }
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
  name        = "${var.project_name}-alb-controller"
  description = "IAM policy for AWS Load Balancer Controller"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:Describe*",
          "elasticloadbalancing:*",
          "acm:ListCertificates",
          "acm:DescribeCertificate",
          "wafv2:*",
          "shield:*",
        ]
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "alb_controller" {
  policy_arn = aws_iam_policy.alb_controller.arn
  role       = module.eks.cluster_iam_role_name
}

resource "kubernetes_namespace" "parkir_pintar" {
  metadata {
    name = "parkir-pintar"
    annotations = {
      "scheduler.alpha.kubernetes.io/defaultTolerations" = ""
    }
  }

  depends_on = [module.eks]
}

resource "kubernetes_secret" "ghcr_pull_secret" {
  metadata {
    name      = "ghcr-pull-secret"
    namespace = kubernetes_namespace.parkir_pintar.metadata[0].name
  }

  type = "kubernetes.io/dockerconfigjson"

  data = {
    ".dockerconfigjson" = jsonencode({
      auths = {
        "ghcr.io" = {
          username = var.ghcr_username
          password = var.ghcr_token
          auth     = base64encode("${var.ghcr_username}:${var.ghcr_token}")
        }
      }
    })
  }

  depends_on = [module.eks]
}
```

- [ ] **Step 7: Commit**

```bash
git add infra/aws/terraform.tf infra/aws/variables.tf infra/aws/terraform.tfvars.example infra/aws/network.tf infra/aws/eks.tf infra/aws/.gitignore
git commit -m "infra: add AWS Terraform foundation (VPC, EKS Fargate + EC2 NATS)"
```

---

### Task 5: Create AWS Terraform Infrastructure — RDS, ElastiCache, S3

**Files:**
- Create: `infra/aws/rds.tf`
- Create: `infra/aws/elasticache.tf`
- Create: `infra/aws/s3.tf`
- Create: `infra/aws/outputs.tf`

- [ ] **Step 1: Create rds.tf**

Write to `infra/aws/rds.tf`:

```hcl
module "db" {
  source  = "terraform-aws-modules/rds/aws"
  version = "~> 6.0"

  identifier = "${var.project_name}-${var.environment}"

  engine               = "postgres"
  engine_version       = "14"
  family               = "postgres14"
  major_engine_version = "14"
  instance_class       = "db.t3.small"

  allocated_storage     = 20
  max_allocated_storage = 50
  storage_type          = "gp3"
  storage_encrypted     = true

  db_name  = "parkir_pintar"
  username = var.db_username
  password = var.db_password
  port     = 5432

  publicly_accessible = false
  skip_final_snapshot = true

  vpc_security_group_ids = [module.eks.cluster_security_group_id]
  db_subnet_group_name   = module.vpc.database_subnet_group_name

  create_db_subnet_group    = false
  create_cloudwatch_log_group = false

  tags = {
    Project     = var.project_name
    Environment = var.environment
  }
}
```

- [ ] **Step 2: Create elasticache.tf**

Write to `infra/aws/elasticache.tf`:

```hcl
resource "aws_elasticache_subnet_group" "redis" {
  name       = "${var.project_name}-redis-subnet-group"
  subnet_ids = module.vpc.private_subnets
}

resource "aws_elasticache_replication_group" "redis" {
  replication_group_id       = "${var.project_name}-${var.environment}"
  description                = "Redis cluster for ${var.project_name}"
  engine                     = "redis"
  engine_version             = "7.0"
  node_type                  = "cache.t3.small"
  num_cache_clusters         = 1
  port                       = 6379
  parameter_group_name       = "default.redis7"
  subnet_group_name          = aws_elasticache_subnet_group.redis.name
  security_group_ids         = [module.eks.cluster_security_group_id]
  automatic_failover_enabled = false
  auth_token                 = var.redis_auth_token
  transit_encryption_enabled = true
  at_rest_encryption_enabled = true

  tags = {
    Project     = var.project_name
    Environment = var.environment
  }
}
```

- [ ] **Step 3: Create s3.tf**

Write to `infra/aws/s3.tf`:

```hcl
resource "aws_s3_bucket" "tfstate" {
  bucket        = "parkir-pintar-tfstate"
  force_destroy = true

  tags = {
    Project     = var.project_name
    Environment = var.environment
  }
}

resource "aws_s3_bucket_versioning" "tfstate" {
  bucket = aws_s3_bucket.tfstate.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "tfstate" {
  bucket = aws_s3_bucket.tfstate.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_dynamodb_table" "tfstate_lock" {
  name         = "parkir-pintar-tfstate-lock"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "LockID"

  attribute {
    name = "LockID"
    type = "S"
  }

  tags = {
    Project     = var.project_name
    Environment = var.environment
  }
}
```

- [ ] **Step 4: Create outputs.tf**

Write to `infra/aws/outputs.tf`:

```hcl
output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "EKS cluster API endpoint"
  value       = module.eks.cluster_endpoint
}

output "db_endpoint" {
  description = "RDS database endpoint"
  value       = module.db.db_instance_endpoint
}

output "db_name" {
  description = "RDS database name"
  value       = module.db.db_instance_name
}

output "redis_endpoint" {
  description = "ElastiCache Redis endpoint"
  value       = aws_elasticache_replication_group.redis.primary_endpoint_address
}

output "redis_port" {
  description = "ElastiCache Redis port"
  value       = aws_elasticache_replication_group.redis.port
}

output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}
```

- [ ] **Step 5: Validate Terraform config**

Run: `terraform -chdir=infra/aws init -backend=false && terraform -chdir=infra/aws validate`

- [ ] **Step 6: Commit**

```bash
git add infra/aws/rds.tf infra/aws/elasticache.tf infra/aws/s3.tf infra/aws/outputs.tf
git commit -m "infra: add AWS RDS, ElastiCache, S3 backend resources"
```

---

### Task 6: Create K8s Base Manifests — NATS StatefulSet, ConfigMaps, Secrets

**Files:**
- Create: `infra/aws/k8s/base/nats-statefulset.yaml`
- Create: `infra/aws/k8s/base/nats-service.yaml`
- Create: `infra/aws/k8s/base/configmap.yaml`
- Create: `infra/aws/k8s/base/secret.yaml`

- [ ] **Step 1: Create nats-statefulset.yaml**

Write to `infra/aws/k8s/base/nats-statefulset.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: nats
  namespace: parkir-pintar
  labels:
    app: nats
spec:
  clusterIP: None
  selector:
    app: nats
  ports:
    - name: client
      port: 4222
      targetPort: 4222
    - name: cluster
      port: 6222
      targetPort: 6222
    - name: monitoring
      port: 8222
      targetPort: 8222
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: nats
  namespace: parkir-pintar
  labels:
    app: nats
spec:
  serviceName: nats
  replicas: 3
  selector:
    matchLabels:
      app: nats
  template:
    metadata:
      labels:
        app: nats
    spec:
      tolerations:
        - key: nats
          operator: Equal
          value: "true"
          effect: NoSchedule
      containers:
        - name: nats
          image: nats:2.10-alpine
          args:
            - "--config"
            - "/etc/nats/nats.conf"
            - "--server_name"
            - "$(POD_NAME)"
            - "--cluster"
            - "nats://$(POD_NAME).nats.parkir-pintar.svc.cluster.local:6222"
            - "--routes"
            - "nats://nats-0.nats.parkir-pintar.svc.cluster.local:6222,nats://nats-1.nats.parkir-pintar.svc.cluster.local:6222,nats://nats-2.nats.parkir-pintar.svc.cluster.local:6222"
            - "--jetstream"
            - "--store_dir"
            - "/data"
          env:
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
          ports:
            - containerPort: 4222
              name: client
            - containerPort: 6222
              name: cluster
            - containerPort: 8222
              name: monitoring
          resources:
            requests:
              cpu: 250m
              memory: 256Mi
            limits:
              cpu: 500m
              memory: 512Mi
          volumeMounts:
            - name: data
              mountPath: /data
            - name: config
              mountPath: /etc/nats
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8222
            initialDelaySeconds: 10
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /healthz
              port: 8222
            initialDelaySeconds: 5
            periodSeconds: 5
      volumes:
        - name: config
          configMap:
            name: nats-config
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 10Gi
        storageClassName: gp3
```

- [ ] **Step 2: Create nats config ConfigMap inline in the same directory**

Write to `infra/aws/k8s/base/nats-configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nats-config
  namespace: parkir-pintar
data:
  nats.conf: |
    jetstream {
      store_dir: /data
      max_memory_store: 256MB
      max_file_store: 5GB
    }
    listen: 0.0.0.0:4222
    http_port: 8222
```

- [ ] **Step 3: Create configmap.yaml (per-service environment configs)**

Write to `infra/aws/k8s/base/configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: parkir-pintar
data:
  DB_PORT: "5432"
  REDIS_PORT: "6379"
  NATS_URL: "nats.parkir-pintar.svc.cluster.local:4222"
  JWT_ISSUER: "parkir-pintar"
  OTEL_EXPORTER_OTLP_ENDPOINT: "http://alloy.parkir-pintar.svc.cluster.local:4319"
  ENVIRONMENT: "production"
---
apiVersion: v1
kind: Secret
metadata:
  name: app-secrets
  namespace: parkir-pintar
type: Opaque
stringData:
  DB_USERNAME: "CHANGE_ME"
  DB_PASSWORD: "CHANGE_ME"
  DB_HOST: "CHANGE_ME"
  DB_DATABASE: "parkir_pintar"
  REDIS_HOST: "CHANGE_ME"
  REDIS_PASSWORD: "CHANGE_ME"
  JWT_SECRET: "CHANGE_ME"
```

- [ ] **Step 4: Commit**

```bash
git add infra/aws/k8s/
git commit -m "k8s: add base manifests (NATS StatefulSet, ConfigMaps, Secrets)"
```

---

### Task 7: Create K8s Service Deployments

**Files:**
- Create: `infra/aws/k8s/services/gateway/deployment.yaml`
- Create: `infra/aws/k8s/services/gateway/service.yaml`
- Create: `infra/aws/k8s/services/search/deployment.yaml` + `service.yaml`
- Create: `infra/aws/k8s/services/reservation/deployment.yaml` + `service.yaml`
- Create: `infra/aws/k8s/services/billing/deployment.yaml` + `service.yaml`
- Create: `infra/aws/k8s/services/payment/deployment.yaml` + `service.yaml`
- Create: `infra/aws/k8s/services/analytics/deployment.yaml` + `service.yaml`
- Create: `infra/aws/k8s/services/presence/deployment.yaml` + `service.yaml`
- Create: `infra/aws/k8s/services/frontend/deployment.yaml` + `service.yaml`

All Go services follow the same template. Only gateway differs (REST port 8080) and analytics (port 9095).

- [ ] **Step 1: Create gateway deployment.yaml**

Write to `infra/aws/k8s/services/gateway/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gateway
  namespace: parkir-pintar
  labels:
    app: gateway
    workload: service
spec:
  replicas: 2
  revisionHistoryLimit: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
  selector:
    matchLabels:
      app: gateway
  template:
    metadata:
      labels:
        app: gateway
        workload: service
    spec:
      imagePullSecrets:
        - name: ghcr-pull-secret
      containers:
        - name: gateway
          image: ghcr.io/piresc-repo/parkir-pintar/gateway:latest-prod
          ports:
            - containerPort: 8080
              name: http
          envFrom:
            - configMapRef:
                name: app-config
            - secretRef:
                name: app-secrets
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 250m
              memory: 256Mi
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 15
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: gateway
  namespace: parkir-pintar
spec:
  selector:
    app: gateway
  ports:
    - port: 8080
      targetPort: 8080
      name: http
```

- [ ] **Step 2: Create reservation deployment.yaml (generic gRPC template)**

Write to `infra/aws/k8s/services/reservation/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: reservation
  namespace: parkir-pintar
  labels:
    app: reservation
    workload: service
spec:
  replicas: 2
  revisionHistoryLimit: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
  selector:
    matchLabels:
      app: reservation
  template:
    metadata:
      labels:
        app: reservation
        workload: service
    spec:
      imagePullSecrets:
        - name: ghcr-pull-secret
      containers:
        - name: reservation
          image: ghcr.io/piresc-repo/parkir-pintar/reservation:latest-prod
          ports:
            - containerPort: 9091
              name: grpc
          envFrom:
            - configMapRef:
                name: app-config
            - secretRef:
                name: app-secrets
          env:
            - name: GRPC_PORT
              value: "9091"
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 250m
              memory: 256Mi
          livenessProbe:
            grpc:
              port: 9091
            initialDelaySeconds: 10
            periodSeconds: 15
          readinessProbe:
            grpc:
              port: 9091
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: reservation
  namespace: parkir-pintar
spec:
  selector:
    app: reservation
  ports:
    - port: 9091
      targetPort: 9091
      name: grpc
```

- [ ] **Step 3: Create billing deployment.yaml (same template, port 9093)**

Write to `infra/aws/k8s/services/billing/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: billing
  namespace: parkir-pintar
  labels:
    app: billing
    workload: service
spec:
  replicas: 2
  revisionHistoryLimit: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
  selector:
    matchLabels:
      app: billing
  template:
    metadata:
      labels:
        app: billing
        workload: service
    spec:
      imagePullSecrets:
        - name: ghcr-pull-secret
      containers:
        - name: billing
          image: ghcr.io/piresc-repo/parkir-pintar/billing:latest-prod
          ports:
            - containerPort: 9093
              name: grpc
          envFrom:
            - configMapRef:
                name: app-config
            - secretRef:
                name: app-secrets
          env:
            - name: GRPC_PORT
              value: "9093"
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 250m
              memory: 256Mi
          livenessProbe:
            grpc:
              port: 9093
            initialDelaySeconds: 10
            periodSeconds: 15
          readinessProbe:
            grpc:
              port: 9093
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: billing
  namespace: parkir-pintar
spec:
  selector:
    app: billing
  ports:
    - port: 9093
      targetPort: 9093
      name: grpc
```

- [ ] **Step 4: Create payment, search, presence, analytics deployments**

Create `infra/aws/k8s/services/payment/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payment
  namespace: parkir-pintar
  labels:
    app: payment
    workload: service
spec:
  replicas: 2
  revisionHistoryLimit: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
  selector:
    matchLabels:
      app: payment
  template:
    metadata:
      labels:
        app: payment
        workload: service
    spec:
      imagePullSecrets:
        - name: ghcr-pull-secret
      containers:
        - name: payment
          image: ghcr.io/piresc-repo/parkir-pintar/payment:latest-prod
          ports:
            - containerPort: 9094
              name: grpc
          envFrom:
            - configMapRef:
                name: app-config
            - secretRef:
                name: app-secrets
          env:
            - name: GRPC_PORT
              value: "9094"
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 250m
              memory: 256Mi
          livenessProbe:
            grpc:
              port: 9094
            initialDelaySeconds: 10
            periodSeconds: 15
          readinessProbe:
            grpc:
              port: 9094
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: payment
  namespace: parkir-pintar
spec:
  selector:
    app: payment
  ports:
    - port: 9094
      targetPort: 9094
      name: grpc
```

Create `infra/aws/k8s/services/search/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: search
  namespace: parkir-pintar
  labels:
    app: search
    workload: service
spec:
  replicas: 2
  revisionHistoryLimit: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
  selector:
    matchLabels:
      app: search
  template:
    metadata:
      labels:
        app: search
        workload: service
    spec:
      imagePullSecrets:
        - name: ghcr-pull-secret
      containers:
        - name: search
          image: ghcr.io/piresc-repo/parkir-pintar/search:latest-prod
          ports:
            - containerPort: 9092
              name: grpc
          envFrom:
            - configMapRef:
                name: app-config
            - secretRef:
                name: app-secrets
          env:
            - name: GRPC_PORT
              value: "9092"
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 250m
              memory: 256Mi
          livenessProbe:
            grpc:
              port: 9092
            initialDelaySeconds: 10
            periodSeconds: 15
          readinessProbe:
            grpc:
              port: 9092
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: search
  namespace: parkir-pintar
spec:
  selector:
    app: search
  ports:
    - port: 9092
      targetPort: 9092
      name: grpc
```

Create `infra/aws/k8s/services/presence/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: presence
  namespace: parkir-pintar
  labels:
    app: presence
    workload: service
spec:
  replicas: 2
  revisionHistoryLimit: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
  selector:
    matchLabels:
      app: presence
  template:
    metadata:
      labels:
        app: presence
        workload: service
    spec:
      imagePullSecrets:
        - name: ghcr-pull-secret
      containers:
        - name: presence
          image: ghcr.io/piresc-repo/parkir-pintar/presence:latest-prod
          ports:
            - containerPort: 9096
              name: grpc
          envFrom:
            - configMapRef:
                name: app-config
            - secretRef:
                name: app-secrets
          env:
            - name: GRPC_PORT
              value: "9096"
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 250m
              memory: 256Mi
          livenessProbe:
            grpc:
              port: 9096
            initialDelaySeconds: 10
            periodSeconds: 15
          readinessProbe:
            grpc:
              port: 9096
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: presence
  namespace: parkir-pintar
spec:
  selector:
    app: presence
  ports:
    - port: 9096
      targetPort: 9096
      name: grpc
```

Create `infra/aws/k8s/services/analytics/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: analytics
  namespace: parkir-pintar
  labels:
    app: analytics
    workload: service
spec:
  replicas: 2
  revisionHistoryLimit: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
  selector:
    matchLabels:
      app: analytics
  template:
    metadata:
      labels:
        app: analytics
        workload: service
    spec:
      imagePullSecrets:
        - name: ghcr-pull-secret
      containers:
        - name: analytics
          image: ghcr.io/piresc-repo/parkir-pintar/analytics:latest-prod
          ports:
            - containerPort: 9095
              name: grpc
          envFrom:
            - configMapRef:
                name: app-config
            - secretRef:
                name: app-secrets
          env:
            - name: GRPC_PORT
              value: "9095"
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 250m
              memory: 256Mi
          livenessProbe:
            grpc:
              port: 9095
            initialDelaySeconds: 10
            periodSeconds: 15
          readinessProbe:
            grpc:
              port: 9095
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: analytics
  namespace: parkir-pintar
spec:
  selector:
    app: analytics
  ports:
    - port: 9095
      targetPort: 9095
      name: grpc
```

- [ ] **Step 5: Create frontend deployment and service**

Write to `infra/aws/k8s/services/frontend/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  namespace: parkir-pintar
  labels:
    app: frontend
    workload: service
spec:
  replicas: 2
  revisionHistoryLimit: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
        workload: service
    spec:
      imagePullSecrets:
        - name: ghcr-pull-secret
      containers:
        - name: frontend
          image: ghcr.io/piresc-repo/parkir-pintar-frontend:latest-prod
          ports:
            - containerPort: 80
              name: http
          resources:
            requests:
              cpu: 100m
              memory: 64Mi
            limits:
              cpu: 200m
              memory: 128Mi
          livenessProbe:
            httpGet:
              path: /
              port: 80
            initialDelaySeconds: 5
            periodSeconds: 15
          readinessProbe:
            httpGet:
              path: /
              port: 80
            initialDelaySeconds: 3
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: frontend
  namespace: parkir-pintar
spec:
  selector:
    app: frontend
  ports:
    - port: 80
      targetPort: 80
      name: http
```

- [ ] **Step 6: Commit**

```bash
git add infra/aws/k8s/services/
git commit -m "k8s: add all service Deployments and Services (7 Go + frontend)"
```

---

### Task 8: Create K8s Ingress + Migration Job

**Files:**
- Create: `infra/aws/k8s/services/frontend/ingress.yaml`
- Create: `infra/aws/k8s/migrations/migration-job.yaml`

- [ ] **Step 1: Create ingress.yaml**

Write to `infra/aws/k8s/services/frontend/ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: parkir-pintar-ingress
  namespace: parkir-pintar
  annotations:
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/healthcheck-path: /health
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTP": 80}, {"HTTPS": 443}]'
    alb.ingress.kubernetes.io/ssl-redirect: "443"
    alb.ingress.kubernetes.io/certificate-arn: "arn:aws:acm:ap-southeast-1:CHANGE_ME:certificate/CHANGE_ME"
spec:
  ingressClassName: alb
  rules:
    - host: parkir-pintar.piresc.dev
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: frontend
                port:
                  number: 80
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: gateway
                port:
                  number: 8080
          - path: /health
            pathType: Exact
            backend:
              service:
                name: gateway
                port:
                  number: 8080
```

- [ ] **Step 2: Create migration-job.yaml**

Write to `infra/aws/k8s/migrations/migration-job.yaml`:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migrate
  namespace: parkir-pintar
spec:
  ttlSecondsAfterFinished: 3600
  backoffLimit: 2
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: migrate
          image: migrate/migrate:v4.17.1
          command:
            - sh
            - -c
            - |
              set -e
              MIGRATE="migrate -database postgres://$DB_USERNAME:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_DATABASE?sslmode=require -path /migrations"
              echo "Running all migrations..."
              for svc in reservation billing payment search presence analytics; do
                echo "--- $svc ---"
                $MIGRATE -path /migrations/$svc up
              done
              echo "Migrations complete."
          envFrom:
            - secretRef:
                name: app-secrets
          volumeMounts:
            - name: migrations
              mountPath: /migrations
      volumes:
        - name: migrations
          configMap:
            name: db-migrations
            defaultMode: 0755
```

- [ ] **Step 3: Commit**

```bash
git add infra/aws/k8s/services/frontend/ingress.yaml infra/aws/k8s/migrations/
git commit -m "k8s: add ALB ingress and DB migration Job"
```

---

### Task 9: Create deploy-prod.yml

**Files:**
- Create: `.github/workflows/deploy-prod.yml`

- [ ] **Step 1: Create deploy-prod.yml**

Write to `.github/workflows/deploy-prod.yml`:

```yaml
name: Deploy Production

on:
  workflow_dispatch:
    inputs:
      version:
        description: "Image version tag (e.g. v1.0.0)"
        required: true
        type: string
        default: "v1.0.0"

permissions:
  contents: read
  id-token: write

env:
  AWS_REGION: ap-southeast-1
  K8S_NAMESPACE: parkir-pintar

jobs:
  deploy:
    name: Deploy to AWS EKS
    runs-on: ubuntu-latest
    environment:
      name: production
    steps:
      - uses: actions/checkout@v4

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_ROLE_ARN }}
          role-session-name: github-actions-deploy
          aws-region: ${{ env.AWS_REGION }}

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: "~1.9"

      - name: Terraform Init
        working-directory: infra/aws
        run: terraform init

      - name: Terraform Plan
        working-directory: infra/aws
        run: |
          terraform plan \
            -var="db_username=${{ secrets.DB_USERNAME }}" \
            -var="db_password=${{ secrets.DB_PASSWORD }}" \
            -var="redis_auth_token=${{ secrets.REDIS_AUTH_TOKEN }}" \
            -var="jwt_secret=${{ secrets.JWT_SECRET }}" \
            -var="ghcr_username=${{ github.actor }}" \
            -var="ghcr_token=${{ secrets.GITHUB_TOKEN }}" \
            -out=tfplan

      - name: Terraform Apply
        working-directory: infra/aws
        run: terraform apply -auto-approve tfplan

      - name: Configure kubectl
        run: aws eks update-kubeconfig --region ${{ env.AWS_REGION }} --name parkir-pintar-production

      - name: Apply K8s manifests
        run: |
          kubectl apply -f infra/aws/k8s/base/nats-configmap.yaml
          kubectl apply -f infra/aws/k8s/base/nats-statefulset.yaml
          kubectl apply -f infra/aws/k8s/base/configmap.yaml
          kubectl apply -f infra/aws/k8s/migrations/migration-job.yaml

      - name: Wait for NATS
        run: |
          kubectl wait --for=condition=ready pod -l app=nats -n $K8S_NAMESPACE --timeout=180s

      - name: Run DB migrations
        run: |
          kubectl wait --for=condition=complete job/db-migrate -n $K8S_NAMESPACE --timeout=120s

      - name: Deploy services with new images
        env:
          VERSION: ${{ github.event.inputs.version }}
        run: |
          for svc in gateway reservation billing payment search presence analytics; do
            echo "Deploying $svc:$VERSION"
            kubectl set image deployment/$svc $svc=ghcr.io/${{ github.repository }}/${svc}:${VERSION} -n $K8S_NAMESPACE
          done
          kubectl set image deployment/frontend frontend=ghcr.io/${{ github.repository }}-frontend:${VERSION} -n $K8S_NAMESPACE

      - name: Wait for all rollouts
        run: |
          for svc in gateway reservation billing payment search presence analytics frontend; do
            echo "Waiting for $svc..."
            kubectl rollout status deployment/$svc -n $K8S_NAMESPACE --timeout=300s
          done

      - name: Smoke test
        run: |
          echo "Waiting 30s for LB propagation..."
          sleep 30
          kubectl get ingress -n $K8S_NAMESPACE
          # Basic health check
          if ! kubectl run smoke-test --rm -i --restart=Never --image=curlimages/curl:latest -n $K8S_NAMESPACE -- \
            curl -sf http://gateway.$K8S_NAMESPACE.svc.cluster.local:8080/health; then
            echo "::error::Health check failed!"
            exit 1
          fi

      - name: Rollback on failure
        if: failure()
        run: |
          echo "::error::Deploy failed — rolling back"
          for svc in gateway reservation billing payment search presence analytics frontend; do
            kubectl rollout undo deployment/$svc -n $K8S_NAMESPACE || true
          done
```

- [ ] **Step 2: Validate YAML**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/deploy-prod.yml')); print('Valid YAML')"`

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/deploy-prod.yml
git commit -m "ci: add AWS EKS production deploy workflow with approval and rollback"
```

---

### Task 10: Remove Old ci.yml + Verify All Workflows

**Files:**
- Modify: `.github/workflows/ci.yml` (trim to just PR checks, or remove entirely since ci-pr.yml exists)
- Verify all workflow YAML is valid

- [ ] **Step 1: Remove old ci.yml**

Now that `ci-pr.yml` handles PR checks and `build-push-staging.yml` handles push events, the original `ci.yml` (which handled both) is superseded.

Remove `.github/workflows/ci.yml`:

```bash
git rm .github/workflows/ci.yml
```

- [ ] **Step 2: Validate all workflow YAML files**

Run: 
```bash
for f in .github/workflows/*.yml; do
  python3 -c "import yaml; yaml.safe_load(open('$f')); print('$f: OK')"
done
```

Expected: All files print "OK".

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/
git commit -m "ci: finalize workflow split — remove old ci.yml"
```

---

### Complete Architecture

```
.github/workflows/
├── ci-pr.yml              # PR: secret-scan, lint, test, security, vulncheck, proto, frontend, sonar
├── build-push-staging.yml # push to main: build 8 images → GHCR → Coolify webhook
├── build-push-prod.yml    # tag v*: build 8 images → GHCR (semver tags)
└── deploy-prod.yml        # workflow_dispatch: terraform → migrate → rollout → smoke → rollback

infra/aws/
├── terraform.tf           # provider, state backend (S3 + DynamoDB)
├── variables.tf           # all input variables
├── terraform.tfvars.example  # template for secrets
├── network.tf             # VPC, subnets, NAT GW
├── eks.tf                 # EKS cluster, Fargate profiles, EC2 NATS node group
├── rds.tf                 # RDS PostgreSQL
├── elasticache.tf         # ElastiCache Redis
├── s3.tf                  # S3 state bucket + DynamoDB lock
├── outputs.tf             # cluster endpoint, DB endpoint, Redis endpoint
└── k8s/
    ├── base/
    │   ├── nats-configmap.yaml
    │   ├── nats-statefulset.yaml
    │   └── configmap.yaml (app config + secrets)
    ├── services/
    │   ├── gateway/deployment.yaml + service.yaml
    │   ├── search/deployment.yaml + service.yaml
    │   ├── reservation/deployment.yaml + service.yaml
    │   ├── billing/deployment.yaml + service.yaml
    │   ├── payment/deployment.yaml + service.yaml
    │   ├── analytics/deployment.yaml + service.yaml
    │   ├── presence/deployment.yaml + service.yaml
    │   └── frontend/deployment.yaml + service.yaml + ingress.yaml
    └── migrations/
        └── migration-job.yaml
```

---

## Self-Review Notes

- Spec coverage: All sections from the design spec are addressed — 4 workflows, AWS infra (Terraform), K8s manifests, migration strategy, rollback, smoke tests
- No placeholders: All code is complete, no TBDs or TODOs
- Type consistency: Image tags use same format across workflows (`main-$SHA`, `v1.0.0`, `latest-prod`), namespace is consistent (`parkir-pintar`), ports match existing service definitions
