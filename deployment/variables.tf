variable "aws_region" {
  description = "AWS region for the queue. eu-central-2 (Zurich) is an opt-in region: it must be enabled on the account first."
  type        = string
  default     = "eu-central-1" # Frankfurt; switch to eu-central-2 once enabled on the account
}

variable "aws_profile" {
  description = "AWS shared credentials profile to use (null = default chain / AWS_PROFILE env var)"
  type        = string
  default     = null
}

variable "project" {
  description = "Project name, used as a prefix for all resources"
  type        = string
  default     = "volley-manager"
}

variable "queue_name" {
  description = "Name of the notifications queue"
  type        = string
  default     = "volley-manager-notifications"
}

variable "message_retention_seconds" {
  description = "How long SQS keeps unconsumed messages (default 7 days)"
  type        = number
  default     = 604800
}

variable "max_receive_count" {
  description = "Deliveries attempted before a message lands in the dead-letter queue"
  type        = number
  default     = 3
}
