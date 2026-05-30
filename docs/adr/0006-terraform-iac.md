# ADR 0006 — Terraform for Infrastructure-as-Code Targeting AWS EKS

## Status

Accepted

## Context

ParkirPintar's production environment requires managed infrastructure (VPC, EKS cluster, RDS, ElastiCache, IAM roles, NLB) in AWS ap-southeast-3 (Jakarta). We needed a way to:

- Define infrastructure declaratively and version-control it
- Reproduce environments reliably (no manual ClickOps drift)
- Support team collaboration with state locking
- Manage cross-resource dependencies (e.g., subnets → EKS → Fargate profiles)

Alternatives considered:

1. **AWS CloudFormation** — AWS-native, but verbose YAML/JSON, slower iteration, no multi-cloud portability
2. **Pulumi** — Programmatic (Go/TS), but smaller ecosystem, team unfamiliar
3. **CDK (AWS)** — Higher-level abstractions, but generates CloudFormation under the hood, debugging is indirect
4. **Terraform** — Declarative HCL, massive provider ecosystem, state management, plan/apply workflow

## Decision

Use **Terraform** with the AWS provider to manage all production infrastructure. State is stored in S3 with DynamoDB locking.

Infrastructure defined in `infra/aws/`:
- `network.tf` — VPC, 2 AZs, public/private subnets, NAT Gateway
- `eks.tf` — EKS cluster, Fargate profile, EC2 node group, EBS CSI driver
- `rds.tf` — RDS PostgreSQL 14 (db.t3.small, encrypted)
- `elasticache.tf` — ElastiCache Redis 7.0 (cache.t3.small)
- `iam-oidc.tf` — GitHub Actions OIDC provider + deploy IAM role
- `s3.tf` — Terraform state bucket + DynamoDB lock table

Kubernetes manifests (`infra/aws/k8s/`) are applied separately via `kubectl` in CI/CD, not managed by Terraform — keeping the blast radius of `terraform apply` limited to infrastructure-level resources.

## Consequences

### Positive

- **Reproducible infrastructure** — `terraform apply` creates identical environments
- **Plan before apply** — `terraform plan` shows exact changes before execution, reducing risk
- **State locking** — S3 + DynamoDB prevents concurrent modifications
- **CI/CD integration** — GitHub Actions uses OIDC to assume an IAM role, no long-lived credentials
- **Cost visibility** — all resources are enumerable from code (~$320/mo estimated)
- **Team familiarity** — Terraform/HCL is widely known, low onboarding friction

### Negative

- **State file dependency** — losing state requires `terraform import` for every resource
- **Provider lag** — new AWS features may not be available immediately in the Terraform provider
- **HCL limitations** — complex logic (loops, conditionals) is less ergonomic than a general-purpose language
- **Two-tool deploy** — Terraform for infra + kubectl for K8s manifests means two deployment paths to maintain
