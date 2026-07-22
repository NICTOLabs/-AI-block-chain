terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true
  tags = {
    Name = "tender-vpc"
  }
}

resource "aws_subnet" "validator_private" {
  count                   = length(var.validator_subnets)
  vpc_id                  = aws_vpc.main.id
  cidr_block              = var.validator_subnets[count.index]
  availability_zone       = var.availability_zones[count.index]
  map_public_ip_on_launch = false
  tags = {
    Name = "tender-validator-private-${count.index + 1}"
  }
}

resource "aws_subnet" "public" {
  count                   = length(var.public_subnets)
  vpc_id                  = aws_vpc.main.id
  cidr_block              = var.public_subnets[count.index]
  availability_zone       = var.availability_zones[count.index + 1]
  map_public_ip_on_launch = true
  tags = {
    Name = "tender-public-${count.index + 1}"
  }
}

resource "aws_internet_gateway" "igw" {
  vpc_id = aws_vpc.main.id
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.igw.id
  }
}

resource "aws_route_table_association" "public" {
  count          = length(aws_subnet.public)
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

resource "aws_kms_key" "tender" {
  description             = "KMS key for Tender node secrets"
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_security_group" "validator" {
  name   = "tender-validator-sg"
  vpc_id = aws_vpc.main.id

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.bastion_cidr]
  }

  ingress {
    from_port   = 3030
    to_port     = 3030
    protocol    = "tcp"
    cidr_blocks = [var.sentry_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "sentry" {
  name   = "tender-sentry-sg"
  vpc_id = aws_vpc.main.id

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.bastion_cidr]
  }

  ingress {
    from_port   = 3030
    to_port     = 3030
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "rpc" {
  name   = "tender-rpc-sg"
  vpc_id = aws_vpc.main.id

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.bastion_cidr]
  }

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_iam_role" "node_role" {
  name = "tender-node-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
      Effect = "Allow"
      Sid    = ""
    }]
  })
}

resource "aws_iam_role_policy" "kms_access" {
  name = "tender-kms-access"
  role = aws_iam_role.node_role.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "kms:Decrypt",
        "kms:DescribeKey"
      ]
      Resource = aws_kms_key.tender.arn
    }]
  })
}

resource "aws_iam_instance_profile" "node_profile" {
  name = "tender-node-profile"
  role = aws_iam_role.node_role.name
}

resource "aws_instance" "validator" {
  count                  = var.validator_count
  ami                    = var.ami_id
  instance_type          = var.validator_instance_type
  subnet_id              = aws_subnet.validator_private[count.index % length(aws_subnet.validator_private)].id
  iam_instance_profile   = aws_iam_instance_profile.node_profile.name
  vpc_security_group_ids = [aws_security_group.validator.id]
  root_block_device {
    volume_type = "gp3"
    volume_size = 200
    iops        = 3000
    throughput  = 125
  }
  tags = {
    Name = "tender-validator-${count.index + 1}"
    Role = "validator"
  }
}

resource "aws_instance" "sentry" {
  count                  = var.sentry_count
  ami                    = var.ami_id
  instance_type          = var.sentry_instance_type
  subnet_id              = aws_subnet.public[count.index % length(aws_subnet.public)].id
  iam_instance_profile   = aws_iam_instance_profile.node_profile.name
  vpc_security_group_ids = [aws_security_group.sentry.id]
  tags = {
    Name = "tender-sentry-${count.index + 1}"
    Role = "sentry"
  }
}

resource "aws_instance" "rpc" {
  count                  = var.rpc_count
  ami                    = var.ami_id
  instance_type          = var.rpc_instance_type
  subnet_id              = aws_subnet.public[count.index % length(aws_subnet.public)].id
  iam_instance_profile   = aws_iam_instance_profile.node_profile.name
  vpc_security_group_ids = [aws_security_group.rpc.id]
  tags = {
    Name = "tender-rpc-${count.index + 1}"
    Role = "rpc"
  }
}
