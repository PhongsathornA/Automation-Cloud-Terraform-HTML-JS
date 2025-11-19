terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket = "terraform-state-phongsathorn-2025" # <--- ⚠️ แก้ชื่อ Bucket ของคุณตรงนี้!
    key    = "terraform.tfstate"
    region = "ap-southeast-1"
  }
}

provider "aws" {
  region = "ap-southeast-1"
}

data "aws_vpc" "default" {
  default = true
}

# --- Subnet (Dynamic CIDR) ---
resource "aws_subnet" "user_selected_subnet" {
  vpc_id            = data.aws_vpc.default.id
  cidr_block        = "172.31.250.0/24" # <--- ค่านี้จะเปลี่ยนตามที่ User เลือก (Auto/Manual)
  availability_zone = "ap-southeast-1a"

  tags = {
    Name = "Subnet-For-R3-Test"
  }
}

# --- Security Group (Dynamic Name) ---
resource "aws_security_group" "user_custom_sg" {
  name        = "Test-Group_SG" # <--- ชื่อ SG ตามที่ User กรอก
  description = "Security Group managed by Terraform Web Portal"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "SSH"
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

  tags = {
    Name = "Test-Group_SG" # แปะป้ายชื่อให้ตรงกันด้วย
  }
}

resource "aws_instance" "web_server" {
  ami           = "ami-0b3eb051c6c7936e9"
  instance_type = "t3.micro"

  subnet_id                   = aws_subnet.user_selected_subnet.id
  vpc_security_group_ids      = [aws_security_group.user_custom_sg.id]
  associate_public_ip_address = true

  tags = {
    Name    = "R3-Test"
    Project = "Cloud-Automation-Web-Generated"
  }
}
