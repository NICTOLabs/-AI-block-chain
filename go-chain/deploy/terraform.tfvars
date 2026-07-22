variable "aws_region" {
  description = "AWS region for validator deployment"
  type        = string
  default     = "eu-west-1"
}

variable "validator_count" {
  description = "Number of validator nodes to deploy"
  type        = number
  default     = 7
}

variable "instance_type" {
  description = "EC2 instance type for validators"
  type        = string
  default     = "t3.xlarge"
}

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "mainnet"
}
