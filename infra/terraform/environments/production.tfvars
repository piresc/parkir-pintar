project_id   = "parkir-pintar-prod"
region       = "asia-southeast2"
environment  = "production"
service_name = "parkir-pintar"

# Cloud SQL
db_tier = "db-custom-2-4096"

# Redis
redis_tier           = "STANDARD_HA"
redis_memory_size_gb = 2

# Cloud Run
max_instances   = 10
container_image = "ghcr.io/piresc/parkir-pintar:production"

# Services
nats_url      = "nats://nats-prod:4222"
otel_endpoint = "http://otel-collector:4317"
