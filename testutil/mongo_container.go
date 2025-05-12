//coverage:ignore file

package testutil

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SetupMongoContainer() (testcontainers.Container, string, error) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "mongo:6.0",
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForLog("Waiting for connections").WithStartupTimeout(time.Second * 30),
	}

	mongoC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		log.Fatalf("Failed to start Mongo container: %v", err)
		return nil, "", err
	}

	endpoint, err := mongoC.Endpoint(ctx, "")
	if err != nil {
		return nil, "", err
	}

	uri := fmt.Sprintf("mongodb://%s", endpoint)
	return mongoC, uri, nil
}