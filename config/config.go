package config

import (
    "github.com/ian-kent/gofigure"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "path/filepath"
)

// Config is the payment reconciliation consumer config
type Config struct {
    gofigure                       interface{} `order:"env,flag"`
    BrokerAddr                     []string    `env:"KAFKA_BROKER_ADDR"                             flag:"broker-addr"                                  flagDesc:"Main CH Kafka broker cluster address"`
    PaymentReconciliationGroupName string      `env:"PAYMENT_RECONCILIATION_GROUP_NAME"             flag:"payment-reconciliation-group-name"            flagDesc:"Payment reconciliation consumer group name"`
    PaymentProcessedTopic          string      `env:"PAYMENT_PROCESSED_TOPIC"                       flag:"payment-processed-topic"                      flagDesc:"Payment processed topic"`
    ZookeeperChroot                string      `env:"KAFKA_ZOOKEEPER_CHROOT"                        flag:"zookeeper-chroot"                             flagDesc:"Main CH Zookeeper chroot"`
    ZookeeperURL                   string      `env:"KAFKA_ZOOKEEPER_ADDR"                          flag:"zookeeper-addr"                               flagDesc:"Main CH Zookeeper address"`
    RetryThrottleRate              int         `env:"RETRY_THROTTLE_RATE_SECONDS"                   flag:"retry-throttle-rate-seconds"                  flagDesc:"Retry throttle rate seconds"`
    MaxRetryAttempts               int         `env:"MAXIMUM_RETRY_ATTEMPTS"                        flag:"max-retry-attemps"                            flagDesc:"Maximum retry attempts"`
    IsErrorConsumer                bool        `env:"IS_ERROR_QUEUE_CONSUMER"                       flag:"is-error-queue-consumer"                      flagDesc:"Set this flag if it is an error queue consumer"`
    ChsAPIKey                      string      `env:"CHS_API_KEY"                                   flag:"chs-api-key"                                  flagDesc:"API access key"`
    SchemaRegistryURL              string      `env:"SCHEMA_REGISTRY_URL"                           flag:"schema-registry-url"                          flagDesc:"Schema registry url"`
    PaymentsAPIURL                 string      `env:"PAYMENTS_API_URL"                              flag:"payments-api-url"                             flagDesc:"Base URL for the Payment Service API"`
    MongoDBURL                     string      `env:"MONGODB_URL"                                   flag:"mongodb-url"                                  flagDesc:"MongoDB server URL"`
    Database                       string      `env:"RECONCILIATION_MONGODB_DATABASE"               flag:"mongodb-database"                             flagDesc:"MongoDB database for data"`
    TransactionsCollection         string      `env:"MONGODB_PAYMENT_REC_TRANSACTIONS_COLLECTION"   flag:"mongodb-payment-rec-transactions-collection"  flagDesc:"MongoDB collection for payment transactions data"`
    ProductsCollection             string      `env:"MONGODB_PAYMENT_REC_PRODUCTS_COLLECTION"       flag:"mongodb-payment-rec-products-collection"      flagDesc:"MongoDB collection for payment products data"`
}

type ProductMap struct {
    Codes map[string]int `yaml:"product_code"`
}

// ServiceConfig returns a ServiceConfig interface for Config
func (c Config) ServiceConfig() ServiceConfig {
    return ServiceConfig{c}
}

// ServiceConfig wraps Config to implement service.Config
type ServiceConfig struct {
    Config
}

// Namespace implements service.Config.Namespace
func (c *Config) Namespace() string {
    return "payment-reconciliation-consumer"
}

func (c *Config) GetProductMap() (*ProductMap, error) {
    filename, _ := filepath.Abs("config/productCode.yml")
    yamlFile, err := ioutil.ReadFile(filename)
    if err != nil {
        return nil, err
    }

    var productCode *ProductMap

    err = yaml.Unmarshal(yamlFile, &productCode)
    if err != nil {
        return nil, err
    }
    return productCode, nil
}

var cfg *Config

// Get configures the application and returns the configuration
func Get() (*Config, error) {
    if cfg != nil {
        return cfg, nil
    }
    cfg = &Config{
        PaymentProcessedTopic:          "",
        PaymentReconciliationGroupName: "payment-reconciliation-consumer-group",
        ZookeeperChroot:                "",
        RetryThrottleRate:              10,
        MaxRetryAttempts:               6,
    }
    err := gofigure.Gofigure(cfg)
    if err != nil {
        return nil, err
    }
    return cfg, nil
}