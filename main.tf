
terraform {
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 5.0" }
  }
  backend "s3" {
    bucket = "terraform-state-phongsathorn-2025" # <--- ⚠️ แก้ชื่อ Bucket ให้ถูก
    key    = "dev-terraform.tfstate"
    region = "ap-southeast-1"
  }
}

provider "aws" { region = "ap-southeast-1" }
data "aws_vpc" "default" { default = true }

resource "aws_subnet" "sub_a" {
  vpc_id            = data.aws_vpc.default.id
  cidr_block        = "172.31.201.0/24"
  availability_zone = "ap-southeast-1a"
  tags              = { Name = "Subnet-A-Test-DB2" }
}
resource "aws_subnet" "sub_b" {
  vpc_id            = data.aws_vpc.default.id
  cidr_block        = "172.31.202.0/24"
  availability_zone = "ap-southeast-1b"
  tags              = { Name = "Subnet-B-Test-DB2" }
}

resource "aws_security_group" "alb_sg" {
  name   = "Test-DB2"
  vpc_id = data.aws_vpc.default.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }


  ingress {
    description = "Database Port"
    from_port   = 3306
    to_port     = 3306
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

resource "aws_lb" "app_lb" {
  name               = "alb-Test-DB2"
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb_sg.id]
  subnets            = [aws_subnet.sub_a.id, aws_subnet.sub_b.id]
}

resource "aws_lb_target_group" "app_tg" {
  name     = "tg-Test-DB2"
  port     = 80
  protocol = "HTTP"
  vpc_id   = data.aws_vpc.default.id
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

resource "aws_launch_template" "app_lt" {
  name_prefix   = "lt-Test-DB2"
  image_id      = "ami-0b3eb051c6c7936e9"
  instance_type = "t3.micro"

  network_interfaces {
    associate_public_ip_address = true
    security_groups             = [aws_security_group.alb_sg.id]
  }

  user_data = base64encode(<<-EOF
              #!/bin/bash
              dnf update -y
              
              
              # Install Nginx
              dnf install -y nginx
              systemctl start nginx
              systemctl enable nginx
              
              # 👇 ส่วนนี้แหละครับที่ดึงเลข ID มาโชว์ 👇
              TOKEN=$(curl -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")
              INSTANCE_ID=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" -s http://169.254.169.254/latest/meta-data/instance-id)
              AZ=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" -s http://169.254.169.254/latest/meta-data/placement/availability-zone)
              
              # เขียนไฟล์ HTML
              cat <<HTML > /usr/share/nginx/html/index.html
              <!DOCTYPE html>
              <html>
              <head>
                  <style>
                      body { font-family: sans-serif; text-align: center; padding-top: 50px; background: #f4f4f4; }
                      .container { background: white; padding: 40px; border-radius: 10px; display: inline-block; box-shadow: 0 4px 15px rgba(0,0,0,0.1); }
                      h1 { color: #2c3e50; }
                      .info { color: #e67e22; font-weight: bold; font-size: 1.2em; }
                      .zone { color: #2980b9; font-weight: bold; }
                  </style>
              </head>
              <body>
                  <div class="container">
                      <h1>Hello from Test-DB2</h1>
                      <p>Served by Instance ID: <span class="info">$INSTANCE_ID</span></p>
                      <p>Availability Zone: <span class="zone">$AZ</span></p>
                      <hr>
                      <small>Deployed via Terraform & Go</small>
                  </div>
              </body>
              </html>
              HTML
              

              
              # Install MariaDB
              dnf install -y mariadb105-server
              systemctl start mariadb
              systemctl enable mariadb
              mysql -e "CREATE DATABASE my_app_db;"
              mysql -e "CREATE USER 'admin'@'%' IDENTIFIED BY 'Pass1234!';"
              mysql -e "GRANT ALL PRIVILEGES ON *.* TO 'admin'@'%';"
              
              EOF
  )
}

resource "aws_autoscaling_group" "app_asg" {
  desired_capacity    = 2
  max_size            = 2
  min_size            = 2
  vpc_zone_identifier = [aws_subnet.sub_a.id, aws_subnet.sub_b.id]
  target_group_arns   = [aws_lb_target_group.app_tg.arn]
  launch_template {
    id      = aws_launch_template.app_lt.id
    version = "$Latest"
  }
}

output "alb_dns_name" {
  value = "http://${aws_lb.app_lb.dns_name}"
}
