output "db_endpoint" {
  value       = aws_db_instance.postgres.endpoint
  description = "The database connection endpoint"
}

output "db_address" {
  value       = aws_db_instance.postgres.address
  description = "The database hostname address"
}

output "db_port" {
  value       = aws_db_instance.postgres.port
  description = "The database connection port"
}
