# payment-reconciliation-consumer
Consumer to allow for reconciliation of payments

## Handling 410 (Gone) Resources
The `payment-reconciliation-consumer` contains handling for situations where a 410 (Gone) status code is returned by the Payments API - with three scenarios available:

* Skip all messages where a 410 (Gone) status code is received
* Do not skip any messages where a 410 (Gone) status code is received
* Only skip messages which relate to a given payment ID, where a 410 (Gone) status code is received

These scenarios can be configured via the `SKIP_GONE_RESOURCE` and `SKIP_GONE_RESOURCE_ID` environment variables with the following configurations.

* `SKIP_GONE_RESOURCE=true` and `SKIP_GONE_RESOURCE_ID` is unset - skip all messages.
* `SKIP_GONE_RESOURCE=false` - do not skip any messages - the value of `SKIP_GONE_RESOURCE_ID` is ignored if one is set.
* `SKIP_GONE_RESOURCE=true` and `SKIP_GONE_RESOURCE_ID=<payment_id>` - only skip messages which receive a 410 gone and match the given payment id.


## Docker support

Pull image from private CH registry by running `docker pull 169942020521.dkr.ecr.eu-west-1.amazonaws.com/local/payment-reconciliation-consumer:latest` command or run the following steps to build image locally:

1. `export SSH_PRIVATE_KEY_PASSPHRASE='[your SSH key passhprase goes here]'` (optional, set only if SSH key is passphrase protected)
2. `DOCKER_BUILDKIT=0 docker build --build-arg SSH_PRIVATE_KEY="$(cat ~/.ssh/id_rsa)" --build-arg SSH_PRIVATE_KEY_PASSPHRASE -t 169942020521.dkr.ecr.eu-west-1.amazonaws.com/local/payment-reconciliation-consumer:latest .`

## Terraform ECS
### What does this code do?
The code present in this repository is used to define and deploy a dockerised container in AWS ECS.
This is done by calling a [module](https://github.com/companieshouse/terraform-modules/tree/main/aws/ecs) from terraform-modules. Application specific attributes are injected and the service is then deployed using Terraform via the CICD platform 'Concourse'.
Application specific attributes | Value                                | Description
:---------|:-----------------------------------------------------------------------------|:-----------
**ECS Cluster**        | payments-service                                     | ECS cluster (stack) the service belongs to
**Load balancer** | N/A (consumer service) | The load balancer that sits in front of the service
**Concourse pipeline**     |[Pipeline link](https://ci-platform.companieshouse.gov.uk/teams/team-development/pipelines/payment-reconciliation-consumer) <br> [Pipeline code](https://github.com/companieshouse/ci-pipelines/blob/master/pipelines/ssplatform/team-development/payment-reconciliation-consumer)                               | Concourse pipeline link in shared services
### Contributing
- Please refer to the [ECS Development and Infrastructure Documentation](https://companieshouse.atlassian.net/wiki/spaces/DEVOPS/pages/4390649858/Copy+of+ECS+Development+and+Infrastructure+Documentation+Updated) for detailed information on the infrastructure being deployed.
### Testing
- Ensure the terraform runner local plan executes without issues. For information on terraform runners please see the [Terraform Runner Quickstart guide](https://companieshouse.atlassian.net/wiki/spaces/DEVOPS/pages/1694236886/Terraform+Runner+Quickstart).
- If you encounter any issues or have questions, reach out to the team on the **#platform** slack channel.
### Vault Configuration Updates
- Any secrets required for this service will be stored in Vault. For any updates to the Vault configuration, please consult with the **#platform** team and submit a workflow request.
### Useful Links
- [ECS service config dev repository](https://github.com/companieshouse/ecs-service-configs-dev)
- [ECS service config production repository](https://github.com/companieshouse/ecs-service-configs-production)