variable "vpc_id" {
  type        = string
  description = "The VPC ID where EKS will be deployed"
}

variable "private_subnet_ids" {
  type        = list(string)
  description = "List of private subnet IDs for EKS node groups"
}

variable "environment" {
  type        = string
  description = "Deployment environment"
}

variable "cluster_name" {
  type        = string
  description = "Name of the EKS cluster"
  default     = "yolo-cluster"
}

variable "node_instance_types" {
  type        = list(string)
  description = "EC2 instance types for EKS nodes"
  default     = ["t3.medium"]
}
