#!/bin/bash
# One-time setup for ParkirPintar production on AWS EKS
# Run this ONCE from infra/aws/ after creating terraform.tfvars

set -e

echo "=== Step 1: Terraform Apply (AWS infra) ==="
terraform init
terraform apply -auto-approve

echo ""
echo "=== Step 2: Configure kubectl ==="
aws eks update-kubeconfig --region ap-southeast-3 --name piresc-parkir

echo ""
echo "=== Step 3: Deploy observability stack ==="
kubectl apply -f infra/aws/k8s/observability/namespace.yaml
kubectl apply -f infra/aws/k8s/observability/alloy.yaml
kubectl apply -f infra/aws/k8s/observability/prometheus.yaml
kubectl apply -f infra/aws/k8s/observability/tempo.yaml
kubectl apply -f infra/aws/k8s/observability/loki.yaml
kubectl apply -f infra/aws/k8s/observability/grafana.yaml

echo ""
echo "=== Step 4: Deploy NATS + base config ==="
kubectl apply -f infra/aws/k8s/base/nats-configmap.yaml
kubectl apply -f infra/aws/k8s/base/nats-statefulset.yaml
kubectl apply -f infra/aws/k8s/base/configmap.yaml

echo ""
echo "=== Step 5: Wait for NATS ==="
kubectl wait --for=condition=ready pod -l app=nats -n pirescer-parkir-pintar --timeout=180s

echo ""
echo "=== Step 6: Update secrets with real endpoints ==="
terraform output db_endpoint
terraform output redis_endpoint
echo ""
echo "Run these commands with the values above:"
echo ""
echo "  kubectl create secret generic app-secrets -n pirescer-parkir-pintar \\"
echo "    --from-literal=DB_USERNAME=parkir_admin \\"
echo "    --from-literal=DB_PASSWORD=<your-db-password> \\"
echo "    --from-literal=DB_HOST=<rds-endpoint> \\"
echo "    --from-literal=DB_DATABASE=parkir_pintar \\"
echo "    --from-literal=DB_PORT=5432 \\"
echo "    --from-literal=REDIS_HOST=<redis-endpoint> \\"
echo "    --from-literal=REDIS_PASSWORD=<your-redis-token> \\"
echo "    --from-literal=REDIS_PORT=6379 \\"
echo "    --from-literal=JWT_SECRET=<your-jwt-secret> \\"
echo "    --from-literal=NATS_URL=nats.pirescer-parkir-pintar.svc.cluster.local:4222 \\"
echo "    --dry-run=client -o yaml | kubectl apply -f -"

echo ""
echo "=== Step 7: Run DB migrations ==="
kubectl create configmap db-migrations --from-file=db/migrations/ -n pirescer-parkir-pintar --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f infra/aws/k8s/migrations/migration-job.yaml
kubectl wait --for=condition=complete job/db-migrate -n pirescer-parkir-pintar --timeout=120s

echo ""
echo "=== Step 8: Deploy services + HPA ==="
for svc in gateway reservation billing payment search presence analytics; do
  kubectl apply -f infra/aws/k8s/services/$svc/deployment.yaml
done
kubectl apply -f infra/aws/k8s/autoscaling/hpa.yaml

echo ""
echo "=== Setup complete! ==="
echo ""
echo "Get the gateway public IP:"
echo "  kubectl get svc gateway -n pirescer-parkir-pintar"
echo ""
echo "GitHub Actions role ARN (add to GitHub Secrets as AWS_ROLE_ARN):"
terraform output github_actions_role_arn
