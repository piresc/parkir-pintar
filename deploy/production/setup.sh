#!/bin/bash
# One-time setup for ParkirPintar production on AWS EKS
# Run from: deploy/production/

set -e

echo "=== Step 1: Terraform Apply (AWS infra) ==="
cd terraform
terraform init
terraform apply -auto-approve
cd ..

echo ""
echo "=== Step 2: Configure kubectl ==="
aws eks update-kubeconfig --region ap-southeast-3 --name piresc-parkir

echo ""
echo "=== Step 3: Deploy observability stack ==="
kubectl apply -f kubernetes/observability/namespace.yaml
kubectl apply -f kubernetes/observability/alloy.yaml
kubectl apply -f kubernetes/observability/prometheus.yaml
kubectl apply -f kubernetes/observability/tempo.yaml
kubectl apply -f kubernetes/observability/loki.yaml
kubectl apply -f kubernetes/observability/grafana.yaml

echo ""
echo "=== Step 4: Deploy NATS + base config ==="
kubectl apply -f kubernetes/base/nats-configmap.yaml
kubectl apply -f kubernetes/base/nats-statefulset.yaml
kubectl apply -f kubernetes/base/configmap.yaml

echo ""
echo "=== Step 5: Wait for NATS ==="
kubectl wait --for=condition=ready pod -l app=nats -n pirescer-parkir-pintar --timeout=180s

echo ""
echo "=== Step 6: Update secrets with real endpoints ==="
DB_HOST=$(cd terraform && terraform output -raw db_endpoint | cut -d: -f1)
REDIS_HOST=$(cd terraform && terraform output -raw redis_endpoint)

# Read secrets from terraform.tfvars
DB_PASSWORD=$(grep db_password terraform.tfvars | grep -oP '"\K[^"]+')
REDIS_PASSWORD=$(grep redis_auth_token terraform.tfvars | grep -oP '"\K[^"]+')
JWT_SECRET=$(grep jwt_secret terraform.tfvars | grep -oP '"\K[^"]+')

kubectl create secret generic app-secrets -n pirescer-parkir-pintar \
  --from-literal=DB_USERNAME=parkir_admin \
  --from-literal=DB_PASSWORD="$DB_PASSWORD" \
  --from-literal=DB_HOST="$DB_HOST" \
  --from-literal=DB_DATABASE=pirescer_parkir_pintar \
  --from-literal=DB_PORT=5432 \
  --from-literal=REDIS_HOST="$REDIS_HOST" \
  --from-literal=REDIS_PASSWORD="$REDIS_PASSWORD" \
  --from-literal=REDIS_PORT=6379 \
  --from-literal=JWT_SECRET="$JWT_SECRET" \
  --from-literal=NATS_URL=nats.pirescer-parkir-pintar.svc.cluster.local:4222 \
  --dry-run=client -o yaml | kubectl apply -f -

echo ""
echo "=== Step 7: Run DB migrations ==="
kubectl create configmap db-migrations --from-file=../../../db/migrations/ -n pirescer-parkir-pintar --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f kubernetes/migrations/migration-job.yaml
kubectl wait --for=condition=complete job/db-migrate -n pirescer-parkir-pintar --timeout=120s

echo ""
echo "=== Step 8: Deploy services + HPA ==="
for svc in gateway reservation billing payment search presence analytics; do
  kubectl apply -f kubernetes/services/$svc/deployment.yaml
done
kubectl apply -f kubernetes/autoscaling/hpa.yaml

echo ""
echo "=== Setup complete! ==="
echo ""
echo "Get the gateway public IP:"
echo "  kubectl get svc gateway -n pirescer-parkir-pintar"
echo ""
echo "GitHub Actions role ARN (add to GitHub Secrets as AWS_ROLE_ARN):"
cd terraform && terraform output github_actions_role_arn
