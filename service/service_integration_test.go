package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/companieshouse/chs.go/avro/schema"
	"github.com/companieshouse/chs.go/kafka/resilience"
	"github.com/companieshouse/payment-reconciliation-consumer/config"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
	testutils "github.com/companieshouse/payment-reconciliation-consumer/testutil"
)

type mockSchemaResponse struct {
	Schema string `json:"schema"`
}

func startMockSchemaRegistry(t *testing.T) *httptest.Server {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := mockSchemaResponse{
			Schema: "{\"type\":\"record\",\"name\":\"payment_processed\",\"namespace\":\"payments\",\"fields\":[{\"name\":\"payment_resource_id\",\"type\":\"string\"}, {\"name\":\"refund_id\",\"type\":\"string\"}]}",
		}
		w.Header().Set("Content-Type", "application/vnd.schemaregistry.v1+json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	return httptest.NewServer(handler)
}

func setupKafkaContainer(t *testing.T) *kafka.KafkaContainer {
	t.Helper()

	ctx := context.Background()

	kafkaContainer, err := kafka.Run(ctx, "confluentinc/cp-kafka:7.5.0", kafka.WithClusterID("test-cluster"), testcontainers.WithExposedPorts("9092"))
	require.NoError(t, err)

	t.Cleanup(func() {
		err := kafkaContainer.Terminate(ctx)
		require.NoError(t, err)
	})

	return kafkaContainer
}

func TestNewService(t *testing.T) {
	container, uri, _ := testutils.SetupMongoContainer()
		defer container.Terminate(context.Background())
	t.Parallel()

	ctx := context.Background()

	mockSchemaRegistry := startMockSchemaRegistry(t)
	defer mockSchemaRegistry.Close()
	kafkaContainer := setupKafkaContainer(t)

	host, err := kafkaContainer.Host(ctx)
	require.NoError(t, err)

	mappedPort, err := kafkaContainer.MappedPort(ctx, "9092")
	require.NoError(t, err)

	bootstrapAddr := fmt.Sprintf("%s:%s", host, mappedPort.Port())

	schemaName := "payment-processed"
	schema, err := schema.Get(mockSchemaRegistry.URL, schemaName)
	require.NoError(t, err)
	require.NotNil(t, schema)

	cfg := &config.Config{
		SchemaRegistryURL: mockSchemaRegistry.URL,
		ZookeeperURL:      "localhost:2181",
		BrokerAddr:        []string{bootstrapAddr},
		ZookeeperChroot:   "",
		ChsAPIKey:         "test-api-key",
		PaymentsAPIURL: "http://mock-payments",
		IsErrorConsumer:   false,
		MongoDBURL:        uri,
	}

	retry := &resilience.ServiceRetry{
		MaxRetries:   1,
		ThrottleRate: 1 * time.Millisecond,
	}

	svc, err := New("test-topic", "test-group", cfg, retry)
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.Equal(t, "test-topic-payment-reconciliation-consumer-retry", svc.Topic)
	require.NotNil(t, svc.Producer)
	require.NotNil(t, svc.Consumer)
}