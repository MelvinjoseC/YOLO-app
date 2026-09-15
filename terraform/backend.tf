# Terraform remote state storage in AWS S3 with state locking via DynamoDB
# This ensures distributed team collaboration without state corruption or race conditions.
terraform {
  backend "s3" {
    bucket         = "yolo-terraform-state-melvinjosec"
    key            = "platform/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "yolo-terraform-locks"
  }
}
