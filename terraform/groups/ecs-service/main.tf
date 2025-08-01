provider "aws" {
  region = var.aws_region
}

terraform {
  backend "s3" {
  }
  required_version = "~> 1.3"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.54.0"
    }
    vault = {
      source  = "hashicorp/vault"
      version = "~> 3.18.0"
    }
  }
}

module "secrets" {
  source = "git@github.com:companieshouse/terraform-modules//aws/ecs/secrets?ref=1.0.324"

  name_prefix = "${local.service_name}-${var.environment}"
  environment = var.environment
  kms_key_id  = data.aws_kms_key.kms_key.id
  secrets     = nonsensitive(local.service_secrets)
}

module "ecs-service" {
  source = "git@github.com:companieshouse/terraform-modules//aws/ecs/ecs-service?ref=1.0.324"

  # Environmental configuration
  environment             = var.environment
  aws_region              = var.aws_region
  aws_profile             = var.aws_profile
  vpc_id                  = data.aws_vpc.vpc.id
  ecs_cluster_id          = data.aws_ecs_cluster.ecs_cluster.id
  task_execution_role_arn = data.aws_iam_role.ecs_cluster_iam_role.arn

  # ECS Task container health check
  use_task_container_healthcheck = true
  healthcheck_path               = local.healthcheck_path
  healthcheck_matcher            = local.healthcheck_matcher

  # Docker container details
  docker_registry   = var.docker_registry
  docker_repo       = local.docker_repo
  container_version = var.payment_reconciliation_consumer_version
  container_port    = local.container_port

  # Service configuration
  service_name                         = local.service_name
  name_prefix                          = local.name_prefix
  desired_task_count                   = var.desired_task_count
  max_task_count                       = var.max_task_count
  min_task_count                       = var.min_task_count
  required_cpus                        = var.required_cpus
  required_memory                      = var.required_memory
  service_autoscale_enabled            = var.service_autoscale_enabled
  service_autoscale_target_value_cpu   = var.service_autoscale_target_value_cpu
  service_scaledown_schedule           = var.service_scaledown_schedule
  service_scaleup_schedule             = var.service_scaleup_schedule
  use_capacity_provider                = var.use_capacity_provider
  use_fargate                          = var.use_fargate
  fargate_subnets                      = local.application_subnet_ids
  service_autoscale_scale_in_cooldown  = var.service_autoscale_scale_in_cooldown
  service_autoscale_scale_out_cooldown = var.service_autoscale_scale_out_cooldown
  read_only_root_filesystem            = false
  batch_service                        = true

  # Service environment variable and secret configs
  task_environment          = local.task_environment
  task_secrets              = local.task_secrets
  app_environment_filename  = local.app_environment_filename
  use_set_environment_files = local.use_set_environment_files
}

module "ecs-service-kafka-v2" {
  count  = var.environment == "cidev" ? 1 : 0
  source = "git@github.com:companieshouse/terraform-modules//aws/ecs/ecs-service?ref=1.0.324"

  # Environmental configuration
  environment             = var.environment
  aws_region              = var.aws_region
  aws_profile             = var.aws_profile
  vpc_id                  = data.aws_vpc.vpc.id
  ecs_cluster_id          = data.aws_ecs_cluster.ecs_cluster.id
  task_execution_role_arn = data.aws_iam_role.ecs_cluster_iam_role.arn
  batch_service           = true


  # ECS Task container health check
  use_task_container_healthcheck = true
  healthcheck_path               = local.healthcheck_path
  healthcheck_matcher            = local.healthcheck_matcher

  # Docker container details
  docker_registry   = var.docker_registry
  docker_repo       = local.docker_repo
  container_version = var.payment_reconciliation_consumer_version
  container_port    = local.container_port

  # Service configuration
  service_name                         = local.service_name_v2
  name_prefix                          = local.name_prefix
  desired_task_count                   = var.desired_task_count
  max_task_count                       = var.max_task_count
  min_task_count                       = var.min_task_count
  required_cpus                        = var.required_cpus
  required_memory                      = var.required_memory
  service_autoscale_enabled            = var.service_autoscale_enabled
  service_autoscale_target_value_cpu   = var.service_autoscale_target_value_cpu
  service_autoscale_scale_in_cooldown  = var.service_autoscale_scale_in_cooldown
  service_autoscale_scale_out_cooldown = var.service_autoscale_scale_out_cooldown
  service_scaledown_schedule           = var.service_scaledown_schedule
  service_scaleup_schedule             = var.service_scaleup_schedule
  use_capacity_provider                = var.use_capacity_provider
  use_fargate                          = var.use_fargate
  fargate_subnets                      = local.application_subnet_ids
  read_only_root_filesystem            = false

  # Service environment variable and secret configs
  task_environment          = local.task_environment_v2
  task_secrets              = local.task_secrets
  use_set_environment_files = local.use_set_environment_files
}
