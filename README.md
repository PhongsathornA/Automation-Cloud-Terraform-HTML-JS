# Cloud Automation By Terraform Project
This project creates AWS EC2 using Terraform and GitHub Actions.

## 🏗 Architecture
1. **Frontend:** HTML/JS for User
2. **Backend:** Go (Golang) Process
3. **Infrastructure:** Terraform Create EC2, Security Groups, VPC
4. **Automation:** GitHub Actions for CI/CD Pipeline (Plan & Apply)

## 🛠 Technologies Used
1. **Language:** Go (Golang), HTML5, JavaScript
2. **IaC:** Terraform
3. **Cloud Provider:** AWS (Amazon Web Services)
4. **CI/CD:** GitHub Actions
5. **State Management:** AWS S3 (Remote Backend)

## ⚙️ Prerequisites
Install For Project
* [Go](https://go.dev/dl/) (1.18+)
* [Terraform](https://developer.hashicorp.com/terraform/install)
* AWS Account & Access Keys (IAM)

## 🚀 How to Use (CI/CD Flow)
This project implements a **GitOps** workflow. Infrastructure changes are deployed automatically via GitHub Actions.
1. **Configure Secrets:**
   Ensure your AWS credentials (`AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`) are set in the GitHub Repository **Settings > Secrets and variables > Actions**.
2. **Modify Infrastructure:**
   Edit Terraform files (e.g., `main.tf`) to adjust your infrastructure requirements.
3. **Push Changes:**
   Commit and push your changes to the `main` branch to trigger the pipeline:
   
   ```bash
   git add .
   git commit -m "Update infrastructure config"
   git push origin main
