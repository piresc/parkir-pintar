# ---------------------------------------------------------------------------
# Route 53 Hosted Zone
# ---------------------------------------------------------------------------
resource "aws_route53_zone" "primary" {
  name = var.domain_name

  tags = {
    Project     = var.project_name
    Environment = var.environment
  }
}

# ---------------------------------------------------------------------------
# Output the NS records (needed if domain registrar is outside AWS)
# ---------------------------------------------------------------------------
output "route53_name_servers" {
  description = "Name servers for the hosted zone"
  value       = aws_route53_zone.primary.name_servers
}

output "route53_zone_id" {
  description = "Hosted zone ID"
  value       = aws_route53_zone.primary.zone_id
}
