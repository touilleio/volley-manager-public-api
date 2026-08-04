data "aws_caller_identity" "current" {}

locals {
  notification_queue_arn_prefix = "arn:aws:sqs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:${var.queue_name}*"
  notification_topic_arn_prefix = "arn:aws:sns:${var.aws_region}:${data.aws_caller_identity.current.account_id}:${var.queue_name}*"
  deployment_user_arn_prefix    = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:user/${var.project}-*"
  profile_name                  = "${var.project}-deployer"
}

resource "aws_iam_user" "deployer" {
  name = "${var.project}-tf-deployer"
}

data "aws_iam_policy_document" "deployer" {
  statement {
    sid = "ManageNotificationsQueues"
    actions = [
      "sqs:CreateQueue",
      "sqs:DeleteQueue",
      "sqs:GetQueueUrl",
      "sqs:GetQueueAttributes",
      "sqs:SetQueueAttributes",
      "sqs:ListQueueTags",
      "sqs:TagQueue",
      "sqs:UntagQueue",
    ]
    resources = [local.notification_queue_arn_prefix]
  }

  statement {
    sid = "ListNotificationsQueues"
    actions = [
      "sqs:ListQueues",
    ]
    resources = ["*"]
  }

  statement {
    sid = "ManageNotificationsTopic"
    actions = [
      "sns:CreateTopic",
      "sns:DeleteTopic",
      "sns:GetTopicAttributes",
      "sns:SetTopicAttributes",
      "sns:TagResource",
      "sns:UntagResource",
      "sns:ListTagsForResource",
      "sns:Subscribe",
      "sns:ListSubscriptionsByTopic",
    ]
    resources = [local.notification_topic_arn_prefix]
  }

  statement {
    sid = "ManageNotificationsSubscriptions"
    actions = [
      "sns:GetSubscriptionAttributes",
      "sns:SetSubscriptionAttributes",
      "sns:Unsubscribe",
    ]
    resources = ["*"]
  }

  statement {
    sid = "ListNotificationsTopicsAndSubscriptions"
    actions = [
      "sns:ListSubscriptions",
      "sns:ListTopics",
    ]
    resources = ["*"]
  }

  statement {
    sid = "ManageQueueIamUsers"
    actions = [
      "iam:CreateUser",
      "iam:GetUser",
      "iam:UpdateUser",
      "iam:DeleteUser",
      "iam:ListGroupsForUser",
      "iam:RemoveUserFromGroup",
      "iam:PutUserPolicy",
      "iam:GetUserPolicy",
      "iam:DeleteUserPolicy",
      "iam:CreateAccessKey",
      "iam:ListAccessKeys",
      "iam:DeleteAccessKey",
      "iam:TagUser",
      "iam:UntagUser",
    ]
    resources = [local.deployment_user_arn_prefix]
  }
}

resource "aws_iam_user_policy" "deployer" {
  name   = "deploy-notifications"
  user   = aws_iam_user.deployer.name
  policy = data.aws_iam_policy_document.deployer.json
}

resource "aws_iam_access_key" "deployer" {
  user = aws_iam_user.deployer.name
}
