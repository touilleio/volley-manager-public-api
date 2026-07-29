resource "aws_sqs_queue" "notifications_dlq" {
  name                      = "${var.queue_name}-dlq"
  message_retention_seconds = 1209600 # 14 days
  sqs_managed_sse_enabled   = true
}

resource "aws_sqs_queue" "notifications" {
  name                       = var.queue_name
  message_retention_seconds  = var.message_retention_seconds
  receive_wait_time_seconds  = 20 # long polling
  visibility_timeout_seconds = 30
  sqs_managed_sse_enabled    = true

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.notifications_dlq.arn
    maxReceiveCount     = var.max_receive_count
  })
}

resource "aws_sqs_queue_redrive_allow_policy" "notifications" {
  queue_url = aws_sqs_queue.notifications_dlq.id

  redrive_allow_policy = jsonencode({
    redrivePermission = "byQueue"
    sourceQueueArns   = [aws_sqs_queue.notifications.arn]
  })
}

data "aws_iam_policy_document" "publisher" {
  statement {
    sid       = "PublishToNotificationsQueue"
    actions   = ["sqs:SendMessage"]
    resources = [aws_sqs_queue.notifications.arn]
  }
}

data "aws_iam_policy_document" "consumer" {
  statement {
    sid = "ConsumeFromNotificationsQueue"
    actions = [
      "sqs:ReceiveMessage",
      "sqs:DeleteMessage",
      "sqs:ChangeMessageVisibility",
      "sqs:GetQueueAttributes",
    ]
    resources = [aws_sqs_queue.notifications.arn]
  }

  statement {
    sid       = "InspectDeadLetterQueue"
    actions   = ["sqs:ReceiveMessage", "sqs:GetQueueAttributes"]
    resources = [aws_sqs_queue.notifications_dlq.arn]
  }
}

resource "aws_iam_user" "publisher" {
  name = "${var.project}-sqs-publisher"
}

resource "aws_iam_user_policy" "publisher" {
  name   = "publish-notifications"
  user   = aws_iam_user.publisher.name
  policy = data.aws_iam_policy_document.publisher.json
}

resource "aws_iam_access_key" "publisher" {
  user = aws_iam_user.publisher.name
}

resource "aws_iam_user" "consumer" {
  name = "${var.project}-sqs-consumer"
}

resource "aws_iam_user_policy" "consumer" {
  name   = "consume-notifications"
  user   = aws_iam_user.consumer.name
  policy = data.aws_iam_policy_document.consumer.json
}

resource "aws_iam_access_key" "consumer" {
  user = aws_iam_user.consumer.name
}
