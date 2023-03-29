variable "k8s_namespace" {
  default = "gitlab"
}
variable "k8s_secretrefname" {
  default = "gitlabber-env"
}
variable "gitlab_url" { }
variable "gitlab_token" { }
variable "s3_backup_bucket_name" { }

variable "s3_backup_path" {
  default = "gitlabber-backup"
}

resource "tls_private_key" "clone_key" {
  algorithm = "ED25519"
}

resource "kubernetes_secret" "conf" {
  metadata {
    name = var.k8s_secretrefname
    namespace = var.k8s_namespace
  }

  data = {
    GITLAB_URL = var.gitlab_url
    GITLAB_TOKEN = var.gitlab_token
    S3_BACKUP_BUCKET_NAME = var.s3_backup_bucket_name
    S3_BACKUP_PATH = var.s3_backup_path
    SSH_PRIVATE_KEY = tls_private_key.clone_key.private_key_openssh
  }
}