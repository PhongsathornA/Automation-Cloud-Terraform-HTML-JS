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

# --- 1. NETWORK (ต้องมี 2 Subnet ใน 2 Zone สำหรับ ALB) ---

resource "aws_subnet" "subnet_a" {
  vpc_id                  = data.aws_vpc.default.id
  cidr_block              = "172.31.201.0/24"
  availability_zone       = "ap-southeast-1a"
  map_public_ip_on_launch = true

  tags = { Name = "Subnet-A-Test-ALB-ASG" }
}

resource "aws_subnet" "subnet_b" {
  vpc_id                  = data.aws_vpc.default.id
  cidr_block              = "172.31.202.0/24"
  availability_zone       = "ap-southeast-1b"
  map_public_ip_on_launch = true

  tags = { Name = "Subnet-B-Test-ALB-ASG" }
}

# --- 2. SECURITY GROUP (สำหรับ ALB และ EC2) ---

resource "aws_security_group" "alb_sg" {
  name        = "Test-ALB-ASG"
  description = "Allow Web traffic to ALB"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "HTTP from World"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "Test-ALB-ASG" }
}

# --- 3. LOAD BALANCER (ALB) ---

resource "aws_lb" "app_lb" {
  name               = "alb-Test-ALB-ASG"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb_sg.id]
  subnets            = [aws_subnet.subnet_a.id, aws_subnet.subnet_b.id]

  tags = { Name = "ALB-Test-ALB-ASG" }
}

resource "aws_lb_target_group" "app_tg" {
  name     = "tg-Test-ALB-ASG"
  port     = 80
  protocol = "HTTP"
  vpc_id   = data.aws_vpc.default.id

  health_check {
    path                = "/"
    healthy_threshold   = 2
    unhealthy_threshold = 2
    timeout             = 5
    interval            = 10
  }
}

resource "aws_lb_listener" "front_end" {
  load_balancer_arn = aws_lb.app_lb.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.app_tg.arn
  }
}

# --- 4. LAUNCH TEMPLATE (แม่พิมพ์สำหรับปั๊ม Server) ---

resource "aws_launch_template" "app_lt" {
  name_prefix   = "lt-Test-ALB-ASG"
  image_id      = "ami-0b3eb051c6c7936e9" # Amazon Linux 2023
  instance_type = "t3.micro"

  network_interfaces {
    associate_public_ip_address = true
    security_groups             = [aws_security_group.alb_sg.id]
  }

  # User Data (Script ติดตั้ง Nginx)

  user_data = base64encode(<<-EOF
              #!/bin/bash
              dnf update -y
              dnf install -y nginx
              systemctl start nginx
              systemctl enable nginx
              
              # สร้างหน้าเว็บที่โชว์ Hostname (จะได้รู้ว่าเข้าเครื่องไหน)
              TOKEN=$(curl -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")
              INSTANCE_ID=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" -v http://169.254.169.254/latest/meta-data/instance-id)
              AZ=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" -v http://169.254.169.254/latest/meta-data/placement/availability-zone)

              cat <<HTML > /usr/share/nginx/html/index.html
              <!DOCTYPE html>
              <html>
              <head>
                  <title>Cluster Demo</title>
                  <style>
                      body { font-family: sans-serif; text-align: center; padding-top: 50px; background: #f0f2f5; }
                      .card { background: white; padding: 30px; display: inline-block; border-radius: 10px; box-shadow: 0 4px 10px rgba(0,0,0,0.1); }
                      h1 { color: #2c3e50; }
                      .id { color: #e67e22; font-weight: bold; }
                      .az { color: #2980b9; font-weight: bold; }
                  </style>
              </head>
              <body>
                  <div class="card">
                      <h1>☁️ Load Balanced App</h1>
                      <p>Served by Instance ID: <span class="id">$INSTANCE_ID</span></p>
                      <p>Availability Zone: <span class="az">$AZ</span></p>
                  </div>
              </body>
              </html>
              HTML
              EOF
  )


  tag_specifications {
    resource_type = "instance"
    tags = {
      Name = "Test-ALB-ASG-Node"
    }
  }
}

# --- 5. AUTO SCALING GROUP (โรงงานปั๊ม Server) ---

resource "aws_autoscaling_group" "app_asg" {
  desired_capacity    = 2
  max_size            = 2
  min_size            = 2
  vpc_zone_identifier = [aws_subnet.subnet_a.id, aws_subnet.subnet_b.id]
  target_group_arns   = [aws_lb_target_group.app_tg.arn]

  launch_template {
    id      = aws_launch_template.app_lt.id
    version = "$Latest"
  }
}

# --- OUTPUTS ---

output "load_balancer_dns" {
  description = "Copy ลิงก์นี้ไปเปิดใน Browser"
  value       = "http://${aws_lb.app_lb.dns_name}"
}
