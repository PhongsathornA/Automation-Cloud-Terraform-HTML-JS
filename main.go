package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
)

type FormData struct {
	ServerName   string
	InstanceType string
	Region       string
	SgName       string
	SubnetCIDR   string
	InstallNginx bool // 👇 เพิ่มตัวแปรนี้
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/generate", handleGenerate)

	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Checkbox: ถ้าติ๊กจะได้ค่า "yes", ถ้าไม่ติ๊กจะได้ค่าว่าง ""
	nginxChoice := r.FormValue("installNginx")
	isInstall := false
	if nginxChoice == "yes" {
		isInstall = true
	}

	subnetMode := r.FormValue("subnetMode")
	finalCidr := ""

	if subnetMode == "manual" {
		finalCidr = r.FormValue("customCidr")
		if finalCidr == "" {
			finalCidr = "172.31.250.0/24"
		}
	} else {
		finalCidr = "172.31.250.0/24" 
	}

	data := FormData{
		ServerName:   r.FormValue("serverName"),
		InstanceType: r.FormValue("instanceType"),
		Region:       r.FormValue("region"),
		SgName:       r.FormValue("sgName"),
		SubnetCIDR:   finalCidr,
		InstallNginx: isInstall, // ส่งค่า Boolean ไปที่ Template
	}

	const tfTemplate = `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket = "terraform-state-phongsathorn-2025"  # <--- ⚠️ แก้ชื่อ Bucket ของคุณตรงนี้!
    key    = "terraform.tfstate"
    region = "{{.Region}}"
  }
}

provider "aws" {
  region = "{{.Region}}"
}

data "aws_vpc" "default" {
  default = true
}

resource "aws_subnet" "user_selected_subnet" {
  vpc_id            = data.aws_vpc.default.id
  cidr_block        = "{{.SubnetCIDR}}"
  availability_zone = "{{.Region}}a"
  
  tags = {
    Name = "Subnet-For-{{.ServerName}}"
  }
}

resource "aws_security_group" "user_custom_sg" {
  name        = "{{.SgName}}"
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
    Name = "{{.SgName}}"
  }
}

resource "aws_instance" "web_server" {
  ami           = "ami-0b3eb051c6c7936e9"
  instance_type = "{{.InstanceType}}"
  
  subnet_id              = aws_subnet.user_selected_subnet.id
  vpc_security_group_ids = [aws_security_group.user_custom_sg.id]
  associate_public_ip_address = true

  # 👇👇👇 Logic เลือกติดตั้ง Nginx 👇👇👇
  {{if .InstallNginx}}
  user_data = <<-EOF
              #!/bin/bash
              sudo apt-get update -y
              sudo apt-get install -y nginx
              sudo systemctl start nginx
              sudo systemctl enable nginx
              echo "<h1>☁️ Hello from {{.ServerName}}!</h1><p>Nginx Installed via Automation</p>" > /var/www/html/index.html
              EOF
  
  user_data_replace_on_change = true
  {{end}}
  # 👆👆👆 ถ้าไม่ได้ติ๊ก Checkbox โค้ดส่วนนี้จะหายไปเลย

  tags = {
    Name    = "{{.ServerName}}"
    Project = "Cloud-Automation-Web-Generated"
  }
}

output "server_public_ip" {
  value = aws_instance.web_server.public_ip
}

output "website_url" {
  value = "http://${aws_instance.web_server.public_ip}"
}
`

	tmpl, err := template.New("terraform").Parse(tfTemplate)
	if err != nil {
		http.Error(w, "Error parsing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	file, err := os.Create("main.tf")
	if err != nil {
		http.Error(w, "Error creating file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	err = tmpl.Execute(file, data)
	if err != nil {
		http.Error(w, "Error saving file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Success Message
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<div style="font-family: sans-serif; text-align: center; padding: 40px;">
			<h1 style="color: green;">✅ สร้างไฟล์สำเร็จ!</h1>
			<p>สถานะการติดตั้ง Nginx: <strong>%t</strong></p>
			
			<div style="background: #f8f9fa; padding: 20px; border: 1px solid #ddd; display: inline-block; text-align: left; border-radius: 8px;">
				<code>
				terraform fmt<br>
				git add .<br>
				git commit -m "Config Nginx option"<br>
				git push
				</code>
			</div>
			<br><br>
			<a href="/">⬅️ กลับหน้าแรก</a>
		</div>
	`, isInstall)
	
	fmt.Printf("Generated: Server=%s, InstallNginx=%t\n", data.ServerName, isInstall)
}