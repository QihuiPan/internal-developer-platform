variable "service_name" {
  description = "DNS-safe service name."
  type        = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{1,61}[a-z0-9]$", var.service_name))
    error_message = "service_name must be a lowercase DNS label between 3 and 63 characters."
  }
}

variable "environment" {
  description = "Deployment environment."
  type        = string
  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "environment must be dev, staging, or production."
  }
}

variable "owner" {
  description = "Owning engineering team."
  type        = string
}
