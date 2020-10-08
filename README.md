# payment-reconciliation-consumer
Consumer to allow for reconciliation of payments

## Docker support

Pull image from private CH registry by running `docker pull 169942020521.dkr.ecr.eu-west-1.amazonaws.com/local/payment-reconciliation-consumer:latest` command or run the following steps to build image locally:

1. `export SSH_PRIVATE_KEY_PASSPHRASE='[your SSH key passhprase goes here]'` (optional, set only if SSH key is passphrase protected)
2. `docker build --build-arg SSH_PRIVATE_KEY="$(cat ~/.ssh/id_rsa)" --build-arg SSH_PRIVATE_KEY_PASSPHRASE -t 169942020521.dkr.ecr.eu-west-1.amazonaws.com/local/payment-reconciliation-consumer:latest .`
