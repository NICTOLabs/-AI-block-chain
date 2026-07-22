output "validator_instance_ids" {
  value = aws_instance.validator[*].id
}

output "sentry_instance_ids" {
  value = aws_instance.sentry[*].id
}

output "rpc_instance_ids" {
  value = aws_instance.rpc[*].id
}

output "kms_key_arn" {
  value = aws_kms_key.tender.arn
}
