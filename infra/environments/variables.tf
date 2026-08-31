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
variable "legal_entity_name" {
  type        = string
  description = "Approved registered entity operating Kredit; displayed publicly."
  validation {
    condition     = length(trimspace(var.legal_entity_name)) >= 3
    error_message = "legal_entity_name must contain the approved registered entity name"
  }
}
variable "legal_service_address" {
  type        = string
  description = "Approved address for legal notices; displayed publicly."
  validation {
    condition     = length(trimspace(var.legal_service_address)) >= 10
    error_message = "legal_service_address must contain the approved service address"
  }
}
variable "legal_contact_email" {
  type        = string
  description = "Public legal contact email."
  validation {
    condition     = can(regex("^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$", var.legal_contact_email))
    error_message = "legal_contact_email must be a valid email address"
  }
}
variable "privacy_contact_email" {
  type        = string
  description = "Public privacy contact email."
  validation {
    condition     = can(regex("^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$", var.privacy_contact_email))
    error_message = "privacy_contact_email must be a valid email address"
  }
}
variable "legal_effective_date" {
  type        = string
  description = "Approved legal-document effective date in YYYY-MM-DD format."
  validation {
    condition     = can(regex("^[0-9]{4}-[0-9]{2}-[0-9]{2}$", var.legal_effective_date))
    error_message = "legal_effective_date must use YYYY-MM-DD"
  }
}
variable "terms_version" {
  type    = string
  default = "supplier-terms-v1"
}
variable "privacy_version" {
  type    = string
  default = "privacy-v1"
}
