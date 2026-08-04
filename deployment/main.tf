resource "aws_sns_topic" "notifications" {
  name = var.queue_name
}

resource "aws_sqs_queue" "notifications_dlq" {
  for_each = toset(var.queue_suffixes)

  name                      = "${var.queue_name}-${each.key}-dlq"
  message_retention_seconds = 1209600 # 14 days
  sqs_managed_sse_enabled   = true
}

resource "aws_sqs_queue" "notifications" {
  for_each = toset(var.queue_suffixes)

  name                       = "${var.queue_name}-${each.key}"
  message_retention_seconds  = var.message_retention_seconds
  receive_wait_time_seconds  = 20 # long polling
  visibility_timeout_seconds = 30
  sqs_managed_sse_enabled    = true

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.notifications_dlq[each.key].arn
    maxReceiveCount     = var.max_receive_count
  })
}

resource "aws_sqs_queue_redrive_allow_policy" "notifications" {
  for_each = toset(var.queue_suffixes)

  queue_url = aws_sqs_queue.notifications_dlq[each.key].id

  redrive_allow_policy = jsonencode({
    redrivePermission = "byQueue"
    sourceQueueArns   = [aws_sqs_queue.notifications[each.key].arn]
  })
}

resource "aws_sns_topic_subscription" "notifications" {
  for_each = toset(var.queue_suffixes)

  topic_arn            = aws_sns_topic.notifications.arn
  protocol             = "sqs"
  endpoint             = aws_sqs_queue.notifications[each.key].arn
  raw_message_delivery = true
}

data "aws_iam_policy_document" "queue" {
  for_each = toset(var.queue_suffixes)

  statement {
    sid     = "AllowNotificationsTopic"
    actions = ["sqs:SendMessage"]

    principals {
      type        = "Service"
      identifiers = ["sns.amazonaws.com"]
    }

    resources = [aws_sqs_queue.notifications[each.key].arn]

    condition {
      test     = "ArnEquals"
      variable = "aws:SourceArn"
      values   = [aws_sns_topic.notifications.arn]
    }
  }
}

resource "aws_sqs_queue_policy" "notifications" {
  for_each = toset(var.queue_suffixes)

  queue_url = aws_sqs_queue.notifications[each.key].id
  policy    = data.aws_iam_policy_document.queue[each.key].json
}

data "aws_iam_policy_document" "publisher" {
  statement {
    sid       = "PublishToNotificationsTopic"
    actions   = ["sns:Publish"]
    resources = [aws_sns_topic.notifications.arn]
  }
}

data "aws_iam_policy_document" "consumer" {
  for_each = toset(var.queue_suffixes)

  statement {
    sid = "ConsumeFromNotificationsQueue"
    actions = [
      "sqs:ReceiveMessage",
      "sqs:DeleteMessage",
      "sqs:ChangeMessageVisibility",
      "sqs:GetQueueAttributes",
    ]
    resources = [aws_sqs_queue.notifications[each.key].arn]
  }

  statement {
    sid       = "InspectDeadLetterQueue"
    actions   = ["sqs:ReceiveMessage", "sqs:GetQueueAttributes"]
    resources = [aws_sqs_queue.notifications_dlq[each.key].arn]
  }
}

resource "aws_iam_user" "publisher" {
  name = "${var.project}-sns-publisher"
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
  for_each = toset(var.queue_suffixes)

  name = "${var.project}-sqs-consumer-${each.key}"
}

resource "aws_iam_user_policy" "consumer" {
  for_each = toset(var.queue_suffixes)

  name   = "consume-notifications"
  user   = aws_iam_user.consumer[each.key].name
  policy = data.aws_iam_policy_document.consumer[each.key].json
}

resource "aws_iam_access_key" "consumer" {
  for_each = toset(var.queue_suffixes)

  user = aws_iam_user.consumer[each.key].name
}
