output "sns_topic_arn" {
  description = "ARN of the notifications SNS topic"
  value       = aws_sns_topic.notifications.arn
}

output "queue_urls" {
  description = "URLs of the notifications queues by suffix"
  value       = { for suffix, queue in aws_sqs_queue.notifications : suffix => queue.url }
}

output "queue_arns" {
  description = "ARNs of the notifications queues by suffix"
  value       = { for suffix, queue in aws_sqs_queue.notifications : suffix => queue.arn }
}

output "dlq_urls" {
  description = "URLs of the dead-letter queues by suffix"
  value       = { for suffix, queue in aws_sqs_queue.notifications_dlq : suffix => queue.url }
}

output "publisher_access_key_id" {
  description = "Access key id of the publisher IAM user"
  value       = aws_iam_access_key.publisher.id
}

output "publisher_secret_access_key" {
  description = "Secret access key of the publisher IAM user"
  value       = aws_iam_access_key.publisher.secret
  sensitive   = true
}

output "consumer_access_key_ids" {
  description = "Access key ids of the consumer IAM users by queue suffix"
  value       = { for suffix, access_key in aws_iam_access_key.consumer : suffix => access_key.id }
}

output "consumer_secret_access_keys" {
  description = "Secret access keys of the consumer IAM users by queue suffix"
  value       = { for suffix, access_key in aws_iam_access_key.consumer : suffix => access_key.secret }
  sensitive   = true
}
