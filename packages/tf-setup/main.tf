variable "k8s_namespace" {
  default = "gitlab"
}
variable "k8s_secretrefname" {
  default = "gitlabber-to-s3"
}

variable "gitlab_url" { }

variable "s3_backup_bucket_name" { }

variable "s3_backup_path" {
  default = "gitlabber-backup"
}

variable "gitlab_deploy_key" {}

resource "gitlab_personal_access_token" "main" {
  user_id    = 2
  name       = "Gitlabber-backup"
  expires_at = "2026-03-14"
  scopes = ["api"]
}

resource "kubernetes_secret" "conf" {
  metadata {
    name = var.k8s_secretrefname
    namespace = var.k8s_namespace
  }

  data = {
    GITLAB_URL = var.gitlab_url
    GITLAB_TOKEN = gitlab_personal_access_token.main.token
    S3_BACKUP_BUCKET_NAME = var.s3_backup_bucket_name
    S3_BACKUP_PATH = var.s3_backup_path
    SSH_PRIVATE_KEY = var.gitlab_deploy_key
  }
}



terraform {
  required_providers {
    gitlab = {
      source = "gitlabhq/gitlab"
    }
 
  }

}
