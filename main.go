package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
)

// เพิ่ม field เพื่อรับค่าใหม่ๆ
type FormData struct {
	ServerName   string
	InstanceType string
	Region       string
	SgName       string // ชื่อ Security Group
	SubnetCIDR   string // เลข IP ของ Subnet
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

	// --- Logic การจัดการ Subnet ---
	subnetMode := r.FormValue("subnetMode")
	finalCidr := ""

	if subnetMode == "manual" {
		// ถ้าเลือก Manual ให้เอาค่าที่พิมพ์มาใช้
		finalCidr = r.FormValue("customCidr")
		// กันเหนียว: ถ้าเลือก Manual แต่ลืมพิมพ์ ให้ใช้ค่า Default
		if finalCidr == "" {
			finalCidr = "172.31.250.0/24"
		}
	} else {
		// ถ้าเลือก Auto ให้ระบบเลือกค่ามาตรฐานให้
		finalCidr = "172.31.250.0/24" 
	}

	// เก็บข้อมูลลง Struct เตรียมส่งเข้า Template
	data := FormData{
		ServerName:   r.FormValue("serverName"),
		InstanceType: r.FormValue("instanceType"),
		Region:       r.FormValue("region"),
		SgName:       r.FormValue("sgName"), // รับชื่อ SG
		SubnetCIDR:   finalCidr,             // รับเลข IP ที่ผ่าน Logic แล้ว
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

# --- Subnet (Dynamic CIDR) ---
resource "aws_subnet" "user_selected_subnet" {
  vpc_id            = data.aws_vpc.default.id
  cidr_block        = "{{.SubnetCIDR}}"      # <--- ค่านี้จะเปลี่ยนตามที่ User เลือก (Auto/Manual)
  availability_zone = "{{.Region}}a"
  
  tags = {
    Name = "Subnet-For-{{.ServerName}}"
  }
}

# --- Security Group (Dynamic Name) ---
resource "aws_security_group" "user_custom_sg" {
  name        = "{{.SgName}}"                # <--- ชื่อ SG ตามที่ User กรอก
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
    Name = "{{.SgName}}" # แปะป้ายชื่อให้ตรงกันด้วย
  }
}

resource "aws_instance" "web_server" {
  ami           = "ami-0b3eb051c6c7936e9"
  instance_type = "{{.InstanceType}}"
  
  subnet_id              = aws_subnet.user_selected_subnet.id
  vpc_security_group_ids = [aws_security_group.user_custom_sg.id]
  associate_public_ip_address = true

  tags = {
    Name    = "{{.ServerName}}"
    Project = "Cloud-Automation-Web-Generated"
  }
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

	// แจ้งเตือน Success
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<div style="font-family: sans-serif; text-align: center; padding: 40px;">
			<h1 style="color: green;">✅ สร้างไฟล์สำเร็จ!</h1>
			<p>Config ที่เลือก:</p>
			<ul style="list-style: none;">
				<li><strong>Server:</strong> %s</li>
				<li><strong>Security Group:</strong> %s</li>
				<li><strong>Subnet CIDR:</strong> %s</li>
			</ul>
			
			<div style="background: #f8f9fa; padding: 20px; border: 1px solid #ddd; display: inline-block; text-align: left; border-radius: 8px;">
				<code>
				terraform fmt<br>
				git add .<br>
				git commit -m "Update infra with custom SG and Subnet"<br>
				git push
				</code>
			</div>
			<br><br>
			<a href="/">⬅️ กลับหน้าแรก</a>
		</div>
	`, data.ServerName, data.SgName, data.SubnetCIDR)
	
	fmt.Printf("Generated: Server=%s, SG=%s, Subnet=%s\n", data.ServerName, data.SgName, data.SubnetCIDR)
}