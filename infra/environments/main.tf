locals {
  labels = {
    "app.kubernetes.io/name"        = "kredit"
    "app.kubernetes.io/part-of"     = "kredit"
    "app.kubernetes.io/environment" = var.environment
  }
  common_env = [
    { name = "APP_ENV", value = var.environment },
    { name = "PUBLIC_BASE_URL", value = var.public_base_url },
    { name = "APP_BASE_URL", value = var.public_base_url },
    { name = "OTEL_EXPORTER_OTLP_ENDPOINT", value = var.otel_endpoint }
  ]
}

resource "kubernetes_namespace_v1" "kredit" {
  metadata {
    name   = "${var.namespace}-${var.environment}"
    labels = local.labels
  }
}

resource "kubernetes_service_account_v1" "api" {
  metadata {
    name      = "api"
    namespace = kubernetes_namespace_v1.kredit.metadata[0].name
    labels    = local.labels
  }
  automount_service_account_token = false
}
resource "kubernetes_service_account_v1" "worker" {
  metadata {
    name      = "worker"
    namespace = kubernetes_namespace_v1.kredit.metadata[0].name
    labels    = local.labels
  }
  automount_service_account_token = false
}
resource "kubernetes_service_account_v1" "web" {
  metadata {
    name      = "web"
    namespace = kubernetes_namespace_v1.kredit.metadata[0].name
    labels    = local.labels
  }
  automount_service_account_token = false
}

resource "kubernetes_deployment_v1" "api" {
  metadata {
    name      = "api"
    namespace = kubernetes_namespace_v1.kredit.metadata[0].name
    labels    = merge(local.labels, { component = "api" })
  }
  spec {
    replicas = var.api_replicas
    selector { match_labels = { component = "api" } }
    strategy {
      type = "RollingUpdate"
      rolling_update {
        max_unavailable = 0
        max_surge       = 1
      }
    }
    template {
      metadata { labels = merge(local.labels, { component = "api" }) }
      spec {
        service_account_name = kubernetes_service_account_v1.api.metadata[0].name
        security_context {
          run_as_non_root = true
          seccomp_profile { type = "RuntimeDefault" }
        }
        container {
          name              = "api"
          image             = var.api_image
          image_pull_policy = var.image_pull_policy
          dynamic "env" {
            for_each = local.common_env
            content {
              name  = env.value.name
              value = env.value.value
            }
          }
          env_from { secret_ref { name = var.runtime_secret_name } }
          port { name = "http"
            container_port = 8080 }
          readiness_probe { http_get { path = "/readyz"
              port = "http" } }
          liveness_probe { http_get { path = "/healthz"
              port = "http" } }
          resources {
            requests = { cpu = "200m", memory = "256Mi" }
            limits   = { cpu = "1", memory = "512Mi" }
          }
          security_context {
            allow_privilege_escalation = false
            read_only_root_filesystem  = true
            capabilities { drop = ["ALL"] }
          }
        }
        termination_grace_period_seconds = 30
      }
    }
  }
}

resource "kubernetes_deployment_v1" "worker" {
  metadata {
    name      = "worker"
    namespace = kubernetes_namespace_v1.kredit.metadata[0].name
    labels    = merge(local.labels, { component = "worker" })
  }
  spec {
    replicas = var.worker_replicas
    selector { match_labels = { component = "worker" } }
    template {
      metadata { labels = merge(local.labels, { component = "worker" }) }
      spec {
        service_account_name = kubernetes_service_account_v1.worker.metadata[0].name
        security_context {
          run_as_non_root = true
          seccomp_profile { type = "RuntimeDefault" }
        }
        container {
          name              = "worker"
          image             = var.worker_image
          image_pull_policy = var.image_pull_policy
          dynamic "env" {
            for_each = local.common_env
            content {
              name  = env.value.name
              value = env.value.value
            }
          }
          env_from { secret_ref { name = var.runtime_secret_name } }
          port { name = "health"
            container_port = 8081 }
          liveness_probe {
            http_get { path = "/healthz"
              port = "health" }
            initial_delay_seconds = 15
            period_seconds        = 30
          }
          readiness_probe {
            http_get { path = "/readyz"
              port = "health" }
            initial_delay_seconds = 5
            period_seconds        = 15
          }
          resources {
            requests = { cpu = "200m", memory = "256Mi" }
            limits   = { cpu = "1", memory = "768Mi" }
          }
          security_context {
            allow_privilege_escalation = false
            read_only_root_filesystem  = true
            capabilities { drop = ["ALL"] }
          }
        }
        termination_grace_period_seconds = 60
      }
    }
  }
}

resource "kubernetes_deployment_v1" "web" {
  metadata {
    name      = "web"
    namespace = kubernetes_namespace_v1.kredit.metadata[0].name
    labels    = merge(local.labels, { component = "web" })
  }
  spec {
    replicas = var.web_replicas
    selector { match_labels = { component = "web" } }
    template {
      metadata { labels = merge(local.labels, { component = "web" }) }
      spec {
        service_account_name = kubernetes_service_account_v1.web.metadata[0].name
        security_context {
          run_as_non_root = true
          seccomp_profile { type = "RuntimeDefault" }
        }
        container {
          name              = "web"
          image             = var.web_image
          image_pull_policy = var.image_pull_policy
          port { name = "http"
            container_port = 3000 }
          env { name = "ORIGIN"
            value = var.public_base_url }
          env { name = "APP_ENV"
            value = var.environment }
          env { name = "API_INTERNAL_URL"
            value = "http://api:8080" }
          env { name = "LEGAL_DOCUMENTS_ACTIVE"
            value = "true" }
          env { name = "LEGAL_ENTITY_NAME"
            value = var.legal_entity_name }
          env { name = "LEGAL_SERVICE_ADDRESS"
            value = var.legal_service_address }
          env { name = "LEGAL_CONTACT_EMAIL"
            value = var.legal_contact_email }
          env { name = "PRIVACY_CONTACT_EMAIL"
            value = var.privacy_contact_email }
          env { name = "LEGAL_EFFECTIVE_DATE"
            value = var.legal_effective_date }
          env { name = "TERMS_VERSION"
            value = var.terms_version }
          env { name = "PRIVACY_VERSION"
            value = var.privacy_version }
          readiness_probe { http_get { path = "/"
              port = "http" }
            initial_delay_seconds = 5
            period_seconds        = 15 }
          liveness_probe { http_get { path = "/"
              port = "http" }
            initial_delay_seconds = 20
            period_seconds        = 30 }
          resources {
            requests = { cpu = "100m", memory = "128Mi" }
            limits   = { cpu = "500m", memory = "384Mi" }
          }
          security_context {
            allow_privilege_escalation = false
            read_only_root_filesystem  = true
            capabilities { drop = ["ALL"] }
          }
        }
      }
    }
  }
}

resource "kubernetes_service_v1" "api" {
  metadata { name = "api"
    namespace = kubernetes_namespace_v1.kredit.metadata[0].name }
  spec {
    selector = { component = "api" }
    port { name = "http"
      port = 8080
      target_port = "http" }
  }
}
resource "kubernetes_service_v1" "web" {
  metadata { name = "web"
    namespace = kubernetes_namespace_v1.kredit.metadata[0].name }
  spec {
    selector = { component = "web" }
    port { name = "http"
      port = 80
      target_port = "http" }
  }
}

resource "kubernetes_ingress_v1" "web" {
  metadata {
    name      = "web"
    namespace = kubernetes_namespace_v1.kredit.metadata[0].name
    annotations = { "kubernetes.io/ingress.class" = var.ingress_class }
  }
  spec {
    tls { hosts = [var.host]
      secret_name = var.tls_secret_name }
    rule {
      host = var.host
      http {
        path {
          path      = "/"
          path_type = "Prefix"
          backend { service { name = kubernetes_service_v1.web.metadata[0].name
              port { number = 80 } } }
        }
      }
    }
  }
}

resource "kubernetes_horizontal_pod_autoscaler_v2" "api" {
  metadata { name = "api"
    namespace = kubernetes_namespace_v1.kredit.metadata[0].name }
  spec {
    min_replicas = var.api_replicas
    max_replicas = 10
    scale_target_ref { api_version = "apps/v1"
      kind = "Deployment"
      name = kubernetes_deployment_v1.api.metadata[0].name }
    metric { type = "Resource"
      resource { name = "cpu"
        target { type = "Utilization"
          average_utilization = 70 } } }
  }
}
resource "kubernetes_pod_disruption_budget_v1" "api" {
  metadata { name = "api"
    namespace = kubernetes_namespace_v1.kredit.metadata[0].name }
  spec { min_available = "50%"
    selector { match_labels = { component = "api" } } }
}
resource "kubernetes_network_policy_v1" "default_deny" {
  metadata { name = "default-deny-ingress"
    namespace = kubernetes_namespace_v1.kredit.metadata[0].name }
  spec { pod_selector {}
    policy_types = ["Ingress"] }
}
resource "kubernetes_network_policy_v1" "api_from_web" {
  metadata { name = "api-from-web"
    namespace = kubernetes_namespace_v1.kredit.metadata[0].name }
  spec {
    pod_selector { match_labels = { component = "api" } }
    ingress {
      from { pod_selector { match_labels = { component = "web" } } }
      ports { protocol = "TCP"
        port = "8080" }
    }
    policy_types = ["Ingress"]
  }
}
resource "kubernetes_network_policy_v1" "web_from_ingress" {
  metadata { name = "web-from-ingress"
    namespace = kubernetes_namespace_v1.kredit.metadata[0].name }
  spec {
    pod_selector { match_labels = { component = "web" } }
    ingress {
      from { namespace_selector {
        match_labels = { "kubernetes.io/metadata.name" = var.ingress_namespace } } }
      ports { protocol = "TCP"
        port = "http" }
    }
    policy_types = ["Ingress"]
  }
}
