#!/bin/bash
#
# Start script for filing-notification-sender

APP_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Read brokers from environment and split on comma
IFS=',' read -a BROKERS <<< "${KAFKA_BROKER_ADDR}"

# Ensure we only populate the broker address via application arguments
unset KAFKA_BROKER_ADDR

exec "${APP_DIR}/payment-reconciliation-consumer" $(for b in "${BROKERS[@]}"; do echo -n "-broker-addr=${b} "; done)