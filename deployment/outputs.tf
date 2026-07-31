output "queue_url" {
  description = "URL of the notifications queue (publisher and consumer endpoint)"
  value       = aws_sqs_queue.notifications.url
}

output "queue_arn" {
  description = "ARN of the notifications queue"
  value       = aws_sqs_queue.notifications.arn
}

output "dlq_url" {
  description = "URL of the dead-letter queue"
  value       = aws_sqs_queue.notifications_dlq.url
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

output "consumer_access_key_id" {
  description = "Access key id of the consumer IAM user"
  value       = aws_iam_access_key.consumer.id
}

output "consumer_secret_access_key" {
  description = "Secret access key of the consumer IAM user"
  value       = aws_iam_access_key.consumer.secret
  sensitive   = true
}
