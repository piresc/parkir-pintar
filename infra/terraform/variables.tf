variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for resource deployment"
  type        = string
  default     = "asia-southeast2"
}

variable "environment" {
  description = "Deployment environment (staging or production)"
  type        = string
  default     = "staging"

  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "Environment must be 'staging' or 'production'."
  }
}

variable "service_name" {
  description = "Name of the Cloud Run service"
  type        = string
  default     = "parkir-pintar"
}

variable "container_image" {
  description = "Container image to deploy"
  type        = string
  default     = "ghcr.io/piresc/parkir-pintar:latest"
}

variable "db_tier" {
  description = "Cloud SQL machine tier"
  type        = string
  default     = "db-f1-micro"
}

variable "db_password" {
  description = "Database password for the application user"
  type        = string
  sensitive   = true
}

variable "redis_tier" {
  description = "Memorystore Redis tier (BASIC or STANDARD_HA)"
  type        = string
  default     = "BASIC"
}

variable "redis_memory_size_gb" {
  description = "Redis instance memory size in GB"
  type        = number
  default     = 1
}

variable "max_instances" {
  description = "Maximum number of Cloud Run instances"
  type        = number
  default     = 10
}

variable "nats_url" {
  description = "NATS server URL"
  type        = string
  default     = "nats://nats:4222"
}

variable "otel_endpoint" {
  description = "OpenTelemetry collector OTLP endpoint"
  type        = string
  default     = "http://otel-collector:4317"
}
