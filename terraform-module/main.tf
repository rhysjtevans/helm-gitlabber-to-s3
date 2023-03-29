variable "k8s_namespace" {
  default = "gitlab"
}
variable "k8s_secretrefname" {
  default = "gitlabber-env"
}

variable "gitlab_url" { }
variable "s3_backup_bucket_name" { }

variable "s3_backup_path" {
  default = "gitlabber-backup"
}

resource "tls_private_key" "clone_key" {
  algorithm = "ED25519"
}

resource "gitlab_personal_access_token" "main" {
  user_id    = 2
  name       = "Gitlabber-backup"
  expires_at = "2026-03-14"

  scopes = ["api"]
}

resource "gitlab_user_sshkey" "main" {
  user_id    = 2
  title      = "gitlabber"
  key        = tls_private_key.clone_key.public_key_openssh

  expires_at = "2028-01-21T00:00:00.000Z"
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
    SSH_PRIVATE_KEY = tls_private_key.clone_key.private_key_openssh
  }
}



terraform {
  required_providers {
    gitlab = {
      source = "gitlabhq/gitlab"
    }
 
  }

}
