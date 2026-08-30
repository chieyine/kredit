output "namespace" { value = kubernetes_namespace_v1.kredit.metadata[0].name }
output "api_service" { value = kubernetes_service_v1.api.metadata[0].name }
output "web_url" { value = "https://${var.host}" }
