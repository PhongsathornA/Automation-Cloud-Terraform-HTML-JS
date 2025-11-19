terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # --- 👇 ส่วนที่เพิ่มมา (ต้องแก้ชื่อ Bucket นะครับ!) 👇 ---
  backend "s3" {
    bucket = "terraform-state-phongsathorn-2025"  # <--- ⚠️ แก้ตรงนี้ให้เป็นชื่อ Bucket จริงๆ ของคุณ
    key    = "terraform.tfstate"
    region = "ap-southeast-1"
  }
  # ----------------------------------------------------
}

# 1. บอกว่าจะใช้ Cloud เจ้าไหน (AWS) และโซนไหน (Singapore)
provider "aws" {
  region = "ap-southeast-1"
}

# 2. สร้าง EC2 Instance (Server)
resource "aws_instance" "app_server" {
  ami           = "ami-060e277c0d4cce553" # Ubuntu 20.04 ใน Singapore (Free Tier)
  instance_type = "t2.micro"              # รุ่นประหยัด (Free Tier ใช้ได้)

  tags = {
    Name = "My-Automated-Server"
    Project = "Cloud-Automation-Project"
  }
}