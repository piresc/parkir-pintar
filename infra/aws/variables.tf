variable "aws_region" {
  type    = string
  default = "ap-southeast-1"
}

variable "project_name" {
  type    = string
  default = "parkir-pintar"
}

variable "environment" {
  type    = string
  default = "production"
}

variable "vpc_cidr" {
  type    = string
  default = "10.0.0.0/16"
}

variable "domain_name" {
  type    = string
  default = "parkir-pintar.piresc.dev"
}

variable "db_username" {
  type      = string
  sensitive = true
}

variable "db_password" {
  type      = string
  sensitive = true
}

variable "redis_auth_token" {
  type      = string
  sensitive = true
}

variable "jwt_secret" {
  type      = string
  sensitive = true
}

variable "ghcr_username" {
  type = string
}

variable "ghcr_token" {
  type      = string
  sensitive = true
}

variable "nats_instance_count" {
  type    = number
  default = 3
}

variable "nats_ec2_instance_type" {
  type    = string
  default = "t3.medium"
}

variable "nats_ec2_desired_size" {
  type    = number
  default = 2
}

variable "nats_ec2_max_size" {
  type    = number
  default = 3
}
