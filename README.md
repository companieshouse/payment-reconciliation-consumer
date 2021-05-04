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
