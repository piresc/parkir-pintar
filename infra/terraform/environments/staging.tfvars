project_id   = "parkir-pintar-staging"
region       = "asia-southeast2"
environment  = "staging"
service_name = "parkir-pintar"

# Cloud SQL
db_tier = "db-f1-micro"

# Redis
redis_tier           = "BASIC"
redis_memory_size_gb = 1

# Cloud Run
max_instances   = 3
container_image = "ghcr.io/piresc/parkir-pintar:latest"

# Services
nats_url      = "nats://nats-staging:4222"
otel_endpoint = "http://otel-collector:4317"
