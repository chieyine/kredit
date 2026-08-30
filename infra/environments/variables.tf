variable "environment" {
  type = string
  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "environment must be staging or production"
  }
}
variable "namespace" { type = string
  default = "kredit" }
variable "kubeconfig_path" { type = string
  default = null }
variable "kubeconfig_context" { type = string
  default = null }
variable "api_image" { type = string }
variable "worker_image" { type = string }
variable "web_image" { type = string }
variable "image_pull_policy" { type = string
  default = "IfNotPresent" }
variable "public_base_url" { type = string }
variable "api_replicas" { type = number
  default = 2 }
variable "web_replicas" { type = number
  default = 2 }
variable "worker_replicas" { type = number
  default = 1 }
variable "runtime_secret_name" {
  type        = string
  description = "Pre-created secret containing database, object-store, signing, provider, notification, and approval values."
}
variable "ingress_class" { type = string
  default = "nginx" }
variable "ingress_namespace" {
  type        = string
  default     = "ingress-nginx"
  description = "Namespace hosting the ingress controller; allowed to reach the web service."
}
variable "tls_secret_name" { type = string }
variable "host" { type = string }
variable "otel_endpoint" { type = string }
