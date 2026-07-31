output "deployer_user_name" {
  description = "Name of the deployer IAM user"
  value       = aws_iam_user.deployer.name
}

output "deployer_access_key_id" {
  description = "Access key id of the deployer IAM user"
  value       = aws_iam_access_key.deployer.id
}

output "deployer_secret_access_key" {
  description = "Secret access key of the deployer IAM user"
  value       = aws_iam_access_key.deployer.secret
  sensitive   = true
}

output "aws_profile_name" {
  description = "Profile name to configure locally (aws configure --profile <value>) and reference as aws_profile in deployment/"
  value       = local.profile_name
}
