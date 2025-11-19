package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
)

// โครงสร้างข้อมูลที่จะรับมาจากหน้าเว็บ
type FormData struct {
	ServerName   string
	InstanceType string
	Region       string
}

func main() {
	// หน้าแรกแสดงไฟล์ HTML
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	// เวลากดปุ่มจะวิ่งมาที่นี่
	http.HandleFunc("/generate", handleGenerate)

	fmt.Println("Server started at http://localhost:8080")
	fmt.Println("Opening browser...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. รับค่าจาก Form
	data := FormData{
		ServerName:   r.FormValue("serverName"),
		InstanceType: r.FormValue("instanceType"),
		Region:       r.FormValue("region"),
	}

	// 2. นี่คือ "แม่พิมพ์" (Template) ของ Terraform
	// เราจะเว้นว่าง credentials ไว้ เพราะ GitHub Actions จะจัดการให้
	const tfTemplate = `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket = "terraform-state-phongsathorn-2025"  # <--- ⚠️ แก้ชื่อ Bucket ให้ตรงกับของคุณ!
    key    = "terraform.tfstate"
    region = "{{.Region}}"
  }
}

provider "aws" {
  region = "{{.Region}}"
  # ไม่ต้องใส่ access_key ตรงนี้ (GitHub Actions จะจัดการให้)
}

resource "aws_instance" "web_server" {
  ami           = "ami-0b3eb051c6c7936e9" # Ubuntu 20.04 (Singapore)
  instance_type = "{{.InstanceType}}"

  tags = {
    Name    = "{{.ServerName}}"
    Project = "Cloud-Automation-Web-Generated"
  }
}
`

	// 3. แปลง Template
	tmpl, err := template.New("terraform").Parse(tfTemplate)
	if err != nil {
		http.Error(w, "Error parsing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. สร้างไฟล์ main.tf (ทับของเดิม)
	file, err := os.Create("main.tf")
	if err != nil {
		http.Error(w, "Error creating file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// 5. เขียนข้อมูลลงไฟล์
	err = tmpl.Execute(file, data)
	if err != nil {
		http.Error(w, "Error saving file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 6. ส่งหน้าเว็บตอบกลับ
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<div style="font-family: sans-serif; text-align: center; padding: 50px;">
			<h1 style="color: green;">✅ สร้างไฟล์สำเร็จ! (Generated Success)</h1>
			<p>ตอนนี้ไฟล์ <strong>main.tf</strong> ในเครื่องคุณเปลี่ยนไปแล้ว</p>
			<p>ขั้นตอนต่อไป:</p>
			<ul style="display: inline-block; text-align: left;">
				<li>1. กลับไปที่ VS Code</li>
				<li>2. เปิด Terminal</li>
				<li>3. พิมพ์คำสั่ง: <code>terraform fmt</code></li>
				<li>3. พิมพ์คำสั่ง: <code>git add .</code></li>
				<li>4. พิมพ์คำสั่ง: <code>git commit -m "Update from web"</code></li>
				<li>5. พิมพ์คำสั่ง: <code>git push</code></li>
			</ul>
			<br><br>
			<a href="/" style="padding: 10px 20px; background: #333; color: white; text-decoration: none; border-radius: 5px;">กลับหน้าแรก</a>
		</div>
	`)
	
	fmt.Printf("Generated Terraform for: %s (%s)\n", data.ServerName, data.InstanceType)
}