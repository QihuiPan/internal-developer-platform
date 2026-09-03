terraform {
  required_version = ">= 1.9.0"
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.36"
    }
  }
}

provider "kubernetes" {
  config_path = pathexpand("~/.kube/config")
}

module "payments_notifier" {
  source       = "../../modules/platform-environment"
  service_name = "payments-notifier"
  environment  = "dev"
  owner        = "team-payments"
}
