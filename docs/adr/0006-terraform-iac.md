# 6. Terraform for Infrastructure-as-Code Targeting GCP Cloud Run

## Status

Accepted

## Context

ParkirPintar deploys to Google Cloud Platform, primarily using Cloud Run for service hosting. Infrastructure includes Cloud Run services, Cloud SQL (PostgreSQL), Memorystore (Redis), Artifact Registry, VPC connectors, IAM bindings, and NATS (on GCE or GKE).

Requirements:
- Reproducible deployments across environments (dev, staging, production)
- Environment parity: infrastructure differences between environments should be explicit and minimal
- Drift detection: ability to detect and reconcile manual changes
- Team collaboration: infrastructure changes should be reviewable via pull requests
- GCP-native resource support

Alternatives considered:

1. **Pulumi** — general-purpose IaC with real programming languages (Go, TypeScript), but smaller community for GCP, less mature provider coverage
2. **Raw gcloud CLI scripts** — simple for small setups, but no state management, no drift detection, imperative rather than declarative, hard to reason about at scale
3. **Terraform** — declarative HCL, mature GCP provider, large community, state-based drift detection, well-understood plan/apply workflow

## Decision

We will use **Terraform** with the Google Cloud provider for all infrastructure provisioning and management.

Structure:
```
infra/
├── modules/          # Reusable modules (cloud-run-service, cloud-sql, etc.)
├── environments/
│   ├── dev/
│   ├── staging/
│   └── prod/
└── backend.tf        # Remote state configuration (GCS bucket)
```

State will be stored in a GCS bucket with state locking via Cloud Storage's built-in locking mechanism. Each environment has its own state file.

## Consequences

### Positive

- Declarative: infrastructure is described as desired state, Terraform handles convergence
- Reproducible: `terraform plan` shows exact changes before apply, environments are consistent
- Drift detection: `terraform plan` reveals manual changes made outside of IaC
- Mature GCP support: google and google-beta providers cover all required resources
- PR-based workflow: infrastructure changes are code-reviewed like application code
- Module reuse: common patterns (e.g., Cloud Run service + IAM + VPC connector) are encapsulated

### Negative

- State management overhead: remote state must be secured, backed up, and occasionally repaired (state locks, corruption)
- HCL limitations: not a general-purpose language, complex logic can be awkward (for_each, dynamic blocks)
- Provider lag: new GCP features may not be immediately available in the Terraform provider
- Learning curve: HCL syntax and Terraform workflow (init, plan, apply) require team familiarity
- Blast radius: a misconfigured `terraform apply` on production can cause outages (mitigated by plan review and environment separation)
