variable "aws_region" {
  description = "AWS region for the deployer identity. Must match the region used in deployment/."
  type        = string
  default     = "eu-central-1"
}

variable "aws_profile" {
  description = "AWS shared credentials profile used to run this bootstrap (needs admin-level rights). null = default chain / AWS_PROFILE env var"
  type        = string
  default     = null
}

variable "project" {
  description = "Project name, used as a prefix for all resources"
  type        = string
  default     = "volley-manager"
}

variable "queue_name" {
  description = "Resource-name prefix/base for the notifications SNS topic and SQS queues the deployer may manage. Must match queue_name in deployment/."
  type        = string
  default     = "volley-manager-notifications"
}
