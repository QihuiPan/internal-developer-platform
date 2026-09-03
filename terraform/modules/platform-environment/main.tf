locals {
  namespace = "${var.service_name}-${var.environment}"
  labels = {
    "app.kubernetes.io/name"       = var.service_name
    "platform.demo/environment"    = var.environment
    "platform.demo/owner"          = var.owner
    "platform.demo/managed-by"     = "terraform"
  }
}

resource "kubernetes_namespace_v1" "service" {
  metadata {
    name   = local.namespace
    labels = local.labels
  }
}

resource "kubernetes_resource_quota_v1" "service" {
  metadata {
    name      = "service-budget"
    namespace = kubernetes_namespace_v1.service.metadata[0].name
  }
  spec {
    hard = {
      "requests.cpu"    = var.environment == "production" ? "4" : "2"
      "requests.memory" = var.environment == "production" ? "8Gi" : "4Gi"
      "limits.cpu"      = var.environment == "production" ? "8" : "4"
      "limits.memory"   = var.environment == "production" ? "16Gi" : "8Gi"
      "pods"            = var.environment == "production" ? "30" : "15"
    }
  }
}

resource "kubernetes_network_policy_v1" "default_deny" {
  metadata {
    name      = "default-deny"
    namespace = kubernetes_namespace_v1.service.metadata[0].name
  }
  spec {
    pod_selector {}
    policy_types = ["Ingress", "Egress"]
  }
}
