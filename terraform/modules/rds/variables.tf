variable "vpc_id" {
  type        = string
  description = "The VPC ID where RDS will be deployed"
}

variable "private_subnet_ids" {
  type        = list(string)
  description = "Private subnet IDs for database subnet group"
}

variable "environment" {
  type        = string
  description = "Deployment environment"
}

variable "db_name" {
  type        = string
  description = "Database name"
  default     = "yolo"
}

variable "db_user" {
  type        = string
  description = "Database admin username"
  default     = "postgres"
}

variable "db_password" {
  type        = string
  description = "Database admin password"
  sensitive   = true
  default     = "YoloSecurePassword123!"
}
