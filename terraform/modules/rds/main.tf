resource "aws_security_group" "db" {
  name        = "yolo-${var.environment}-rds-sg"
  description = "Allow inbound database traffic from authorized compute workloads"
  vpc_id      = var.vpc_id

  dynamic "ingress" {
    for_each = length(var.allowed_security_group_ids) > 0 ? [1] : []
    content {
      description     = "PostgreSQL traffic from EKS worker node security groups"
      from_port       = 5432
      to_port         = 5432
      protocol        = "tcp"
      security_groups = var.allowed_security_group_ids
    }
  }

  ingress {
    description = "PostgreSQL traffic from VPC private subnets"
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"]
  }

  egress {
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }

  tags = {
    Name        = "yolo-${var.environment}-db-sg"
    Environment = var.environment
  }
}

resource "aws_db_subnet_group" "main" {
  name       = "yolo-${var.environment}-db-subnet-group"
  subnet_ids = var.private_subnet_ids

  tags = {
    Name        = "yolo-${var.environment}-db-subnet-group"
    Environment = var.environment
  }
}

resource "aws_db_instance" "postgres" {
  identifier                  = "yolo-${var.environment}-db"
  allocated_storage           = 20
  max_allocated_storage       = 100
  db_name                     = var.db_name
  engine                      = "postgres"
  engine_version              = "15.3"
  instance_class              = "db.t3.micro"
  username                    = var.db_user
  password                    = var.db_password
  db_subnet_group_name        = aws_db_subnet_group.main.name
  vpc_security_group_ids      = [aws_security_group.db.id]
  storage_encrypted           = true
  kms_key_id                  = var.kms_key_arn != "" ? var.kms_key_arn : null
  backup_retention_period     = var.environment == "production" ? 14 : 7
  backup_window               = "03:00-04:00"
  maintenance_window          = "Mon:04:00-Mon:05:00"
  auto_minor_version_upgrade  = true
  deletion_protection         = var.environment == "production" ? true : false
  copy_tags_to_snapshot       = true
  skip_final_snapshot         = var.environment == "production" ? false : true

  tags = {
    Name        = "yolo-${var.environment}-db-instance"
    Environment = var.environment
  }
}
