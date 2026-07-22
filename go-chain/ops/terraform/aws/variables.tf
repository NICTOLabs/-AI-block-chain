variable "region" {
  type    = string
  default = "us-east-1"
}

variable "vpc_cidr" {
  type    = string
  default = "10.100.0.0/16"
}

variable "validator_subnets" {
  type    = list(string)
  default = ["10.100.10.0/24", "10.100.11.0/24", "10.100.12.0/24"]
}

variable "public_subnets" {
  type    = list(string)
  default = ["10.100.20.0/24", "10.100.21.0/24"]
}

variable "availability_zones" {
  type    = list(string)
  default = ["us-east-1a", "us-east-1b", "us-east-1c"]
}

variable "bastion_cidr" {
  type    = string
  default = "203.0.113.0/24"
}

variable "sentry_cidr" {
  type    = string
  default = "0.0.0.0/0"
}

variable "ami_id" {
  type    = string
  default = "ami-0c02fb55956c7d316"
}

variable "validator_count" {
  type    = number
  default = 7
}

variable "sentry_count" {
  type    = number
  default = 3
}

variable "rpc_count" {
  type    = number
  default = 2
}

variable "validator_instance_type" {
  type    = string
  default = "m6i.2xlarge"
}

variable "sentry_instance_type" {
  type    = string
  default = "c6i.xlarge"
}

variable "rpc_instance_type" {
  type    = string
  default = "c6i.2xlarge"
}
