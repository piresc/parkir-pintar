module "db" {
  source  = "terraform-aws-modules/rds/aws"
  version = "~> 6.0"

  identifier = "${var.project_name}-${var.environment}"

  engine               = "postgres"
  engine_version       = "14"
  family               = "postgres14"
  major_engine_version = "14"

  instance_class    = "db.t3.small"
  allocated_storage = 20
  max_allocated_storage = 50
  storage_type      = "gp3"
  storage_encrypted = true

  db_name  = "parkir_pintar"
  username = var.db_username
  password = var.db_password
  port     = 5432

  publicly_accessible = false
  skip_final_snapshot = true

  vpc_security_group_ids = [module.eks.cluster_security_group_id]
  db_subnet_group_name   = module.vpc.database_subnet_group_name
  create_db_subnet_group       = false
  create_cloudwatch_log_group  = false

  tags = {
    Project     = var.project_name
    Environment = var.environment
  }
}
