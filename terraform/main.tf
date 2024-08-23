provider "aws" {
  region = "us-east-1" # Change this to your region if needed
}

# Create IAM role for API Gateway to access SQS
resource "aws_iam_role" "api_gateway_role" {
  name = "api-gateway-sqs-role"

  assume_role_policy = jsonencode({
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Principal": {
          "Service": "apigateway.amazonaws.com"
        },
        "Action": "sts:AssumeRole"
      }
    ]
  })
}

# IAM policy for sending messages to the agent queue
resource "aws_iam_role_policy" "agent_queue_policy" {
  name   = "agent-queue-policy"
  role   = aws_iam_role.api_gateway_role.id

  policy = jsonencode({
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "sqs:SendMessage",
          "sqs:SendMessageBatch"
        ],
        "Resource": aws_sqs_queue.agent_queue.arn
      }
    ]
  })
}

# IAM policy for sending messages to the register queue
resource "aws_iam_role_policy" "register_queue_policy" {
  name   = "register-queue-policy"
  role   = aws_iam_role.api_gateway_role.id

  policy = jsonencode({
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "sqs:SendMessage"
        ],
        "Resource": aws_sqs_queue.register_queue.arn
      }
    ]
  })
}

# Create SQS queues
resource "aws_sqs_queue" "agent_queue" {
  name = "agent-queue"
}

resource "aws_sqs_queue" "register_queue" {
  name = "register-queue"
}

# Create API Gateway
resource "aws_apigatewayv2_api" "api" {
  name          = "test-api"
  protocol_type = "HTTP"
}

# Create custom domain for API Gateway
resource "aws_apigatewayv2_domain_name" "custom_domain" {
  domain_name   = "api.ulteriorlabs.io"
  domain_name_configuration {
    certificate_arn = aws_acm_certificate.cert.arn
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }
}

# ACM Certificate for the domain
resource "aws_acm_certificate" "cert" {
  domain_name       = "api.ulteriorlabs.io"
  validation_method = "DNS"
}

# Route 53 record for the API Gateway custom domain
resource "aws_route53_record" "api_record" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = aws_apigatewayv2_domain_name.custom_domain.domain_name
  type    = "A"

  alias {
    name                   = aws_apigatewayv2_domain_name.custom_domain.domain_name_configuration[0].target_domain_name
    zone_id                = aws_apigatewayv2_domain_name.custom_domain.domain_name_configuration[0].hosted_zone_id
    evaluate_target_health = false
  }
}

# Find the existing hosted zone
data "aws_route53_zone" "main" {
  name         = "ulteriorlabs.io"
  private_zone = false
}

# Add API Gateway integration to SQS using the IAM role
resource "aws_apigatewayv2_integration" "sqs_agent_integration" {
  api_id           = aws_apigatewayv2_api.api.id
  integration_type = "AWS_PROXY"
  integration_uri  = aws_sqs_queue.agent_queue.arn
  credentials_arn  = aws_iam_role.api_gateway_role.arn
}

resource "aws_apigatewayv2_integration" "sqs_register_integration" {
  api_id           = aws_apigatewayv2_api.api.id
  integration_type = "AWS_PROXY"
  integration_uri  = aws_sqs_queue.register_queue.arn
  credentials_arn  = aws_iam_role.api_gateway_role.arn
}

# Routes for the API Gateway
resource "aws_apigatewayv2_route" "agent_route" {
  api_id    = aws_apigatewayv2_api.api.id
  route_key = "POST /receive/agent"
  target    = "integrations/${aws_apigatewayv2_integration.sqs_agent_integration.id}"
}

resource "aws_apigatewayv2_route" "register_route" {
  api_id    = aws_apigatewayv2_api.api.id
  route_key = "POST /receive/register"
  target    = "integrations/${aws_apigatewayv2_integration.sqs_register_integration.id}"
}

# API Gateway deployment
resource "aws_apigatewayv2_stage" "test_stage" {
  api_id      = aws_apigatewayv2_api.api.id
  name        = "test"
  description = "Test stage for our API"
  auto_deploy = true
}
