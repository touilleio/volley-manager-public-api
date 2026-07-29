data "aws_caller_identity" "current" {}

locals {
  queue_arns = [
    "arn:aws:sqs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:${var.queue_name}",
    "arn:aws:sqs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:${var.queue_name}-dlq",
  ]
  profile_name = "${var.project}-deployer"
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
      "sqs:GetQueueAttributes",
      "sqs:SetQueueAttributes",
      "sqs:ListQueueTags",
      "sqs:TagQueue",
      "sqs:UntagQueue",
    ]
    resources = local.queue_arns
  }

  statement {
    sid = "ManageQueueIamUsers"
    actions = [
      "iam:CreateUser",
      "iam:GetUser",
      "iam:DeleteUser",
      "iam:PutUserPolicy",
      "iam:GetUserPolicy",
      "iam:DeleteUserPolicy",
      "iam:CreateAccessKey",
      "iam:ListAccessKeys",
      "iam:DeleteAccessKey",
      "iam:TagUser",
      "iam:UntagUser",
    ]
    resources = ["*"]
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
