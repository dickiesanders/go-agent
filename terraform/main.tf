provider "aws" {
  region = var.region
}

terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"  # or whatever version you prefer
    }
  }
}

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

# Variable definitions
variable "account_id" {
  description = "AWS Account ID"
  type        = string
}

variable "region" {
  description = "AWS Region"
  type        = string
}

variable "cloudflare_email" {
  description = "Email used in Cloudflare"
  type        = string
}

variable "cloudflare_api_token" {
  description = "API key for Cloudflare"
  type        = string
}

# ACM Certificate for your domain
resource "aws_acm_certificate" "certificate" {
  domain_name       = "ulteriorlabs.io"
  validation_method = "DNS"

  subject_alternative_names = [
    "api.ulteriorlabs.io"
  ]

  lifecycle {
    create_before_destroy = true
  }
}

# Fetch Cloudflare Zone
data "cloudflare_zones" "zone" {
  filter {
    name = "ulteriorlabs.io"
  }
}

# Create Cloudflare DNS records for ACM validation
# resource "cloudflare_record" "cert_validation" {
#   for_each = {
#     for dvo in aws_acm_certificate.certificate.domain_validation_options : dvo.domain_name => {
#       name    = dvo.resource_record_name
#       type    = dvo.resource_record_type
#       content = dvo.resource_record_value
#     }
#   }

#   zone_id = data.cloudflare_zones.zone.id
#   name    = each.value.name
#   type    = each.value.type
#   content = each.value.content
#   ttl     = 300
# }

# # ACM Certificate Validation
# resource "aws_acm_certificate_validation" "certificate_validation" {
#   certificate_arn         = aws_acm_certificate.certificate.arn
#   validation_record_fqdns = [for record in cloudflare_record.cert_validation : record.hostname]
# }

# API Gateway setup (custom domain with ACM certificate)
resource "aws_apigatewayv2_domain_name" "custom_domain" {
  domain_name = "api.ulteriorlabs.io"
  domain_name_configuration {
    certificate_arn = aws_acm_certificate.certificate.arn
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }
}

# SQS Agent Queue
resource "aws_sqs_queue" "agent_queue" {
  name = "agent-queue"
}

# SQS Register Queue
resource "aws_sqs_queue" "register_queue" {
  name = "register-queue"
}

resource "aws_apigatewayv2_integration" "sqs_agent_integration" {
  api_id              = aws_apigatewayv2_api.api_gw.id
  credentials_arn     = aws_iam_role.api_gw_role.arn
  description         = "SQS Agent Integration"
  integration_type    = "AWS_PROXY"
  integration_subtype = "SQS-SendMessage"

  request_parameters = {
    "QueueUrl"    = "$request.header.queueUrl"
    "MessageBody" = "$request.body.message"
  }

  # payload_format_version = "2.0"
}

# API Gateway integration with SQS (Register)
resource "aws_apigatewayv2_integration" "sqs_register_integration" {
  api_id              = aws_apigatewayv2_api.api_gw.id
  credentials_arn     = aws_iam_role.api_gw_role.arn
  description         = "SQS Register Integration"
  integration_type    = "AWS_PROXY"
  integration_subtype = "SQS-SendMessage"

  request_parameters = {
    "QueueUrl"    = "$request.header.queueUrl"
    "MessageBody" = "$request.body.message"
  }

  # payload_format_version = "2.0"
}

# API Gateway definition
resource "aws_apigatewayv2_api" "api_gw" {
  name          = "Ulterior Labs API Gateway"
  protocol_type = "HTTP"
}

# API Gateway route for Agent Queue
resource "aws_apigatewayv2_route" "agent_route" {
  api_id    = aws_apigatewayv2_api.api_gw.id
  route_key = "POST /agent"
  target    = "integrations/${aws_apigatewayv2_integration.sqs_agent_integration.id}"
}

# API Gateway route for Register Queue
resource "aws_apigatewayv2_route" "register_route" {
  api_id    = aws_apigatewayv2_api.api_gw.id
  route_key = "POST /register"
  target    = "integrations/${aws_apigatewayv2_integration.sqs_register_integration.id}"
}

# API Gateway deployment
resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.api_gw.id
  name        = "$default"
  auto_deploy = true
}

# IAM role for API Gateway to invoke SQS
resource "aws_iam_role" "api_gw_role" {
  name = "api-gw-sqs-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action    = "sts:AssumeRole"
        Effect    = "Allow"
        Principal = {
          Service = "apigateway.amazonaws.com"
        }
      }
    ]
  })
}

# IAM policy to allow API Gateway to send messages to SQS
resource "aws_iam_role_policy" "api_gw_policy" {
  role = aws_iam_role.api_gw_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action   = [
          "sqs:SendMessage",
        ]
        Effect   = "Allow"
        Resource = [aws_sqs_queue.agent_queue.arn, aws_sqs_queue.register_queue.arn]
      }
    ]
  })
}
