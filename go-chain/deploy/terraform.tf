terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

resource "aws_instance" "tender_validator" {
  count         = var.validator_count
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = var.instance_type
  subnet_id     = aws_subnet.validators.id
  vpc_security_group_ids = [aws_security_group.tender_validator.id]

  user_data = base64encode(<<-EOF
              #!/bin/bash
              apt-get update -y
              apt-get install -y docker.io docker-compose git
              git clone https://github.com/NICTOLabs/-AI-block-chain.git /opt/tdr
              cd /opt/tdr/go-chain
              docker build -t tender-node .
              EOF
  )

  tags = {
    Name = "tender-validator-${count.index + 1}"
    Project = "TDR"
    Environment = "mainnet"
  }
}

resource "aws_security_group" "tender_validator" {
  name_prefix = "tender-validator-"
  description = "Security group for TDR validator nodes"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 3030
    to_port     = 3030
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }

  ingress {
    from_port   = 9090
    to_port     = 9090
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }

  ingress {
    from_port   = 22
    to_port     = 22
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
