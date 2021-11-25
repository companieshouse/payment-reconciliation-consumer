package dao

import (
	"context"
	"errors"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payment-reconciliation-consumer/models"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"time"
)

var client *mongo.Client

func getMongoClient(mongoDBURL string) *mongo.Client {
	if client != nil {
		return client
	}

	ctx := context.Background()

	clientOptions := options.Client().ApplyURI(mongoDBURL)
	client, err := mongo.Connect(ctx, clientOptions)

	// Assume the caller of this func cannot handle the case where there is no database connection
	// so the service must crash here as it cannot continue.
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// Check we can connect to the mongodb instance. Failure here should result in a crash.
	pingContext, cancel := context.WithDeadline(ctx, time.Now().Add(5*time.Second))
	err = client.Ping(pingContext, nil)
	if err != nil {
		log.Error(errors.New("ping to mongodb timed out. please check the connection to mongodb and that it is running"))
		os.Exit(1)
	}
	defer cancel()

	log.Info("connected to mongodb successfully")

	return client
}

// MongoDatabaseInterface is an interface that describes the mongodb driver
type MongoDatabaseInterface interface {
	Collection(name string, opts ...*options.CollectionOptions) *mongo.Collection
}

func getMongoDatabase(mongoDBURL, databaseName string) MongoDatabaseInterface {
	return getMongoClient(mongoDBURL).Database(databaseName)
}

// MongoService is an implementation of the Service interface using MongoDB as the backend driver.
type MongoService struct {
	db             MongoDatabaseInterface
	TransactionsCollection string
	ProductsCollection string
	RefundsCollection string
}

// CreateEshuResource will store the eshu file details into the database
func (m *MongoService) CreateEshuResource(eshuResource *models.EshuResourceDao) error {
	collection := m.db.Collection(m.ProductsCollection)
	_, err := collection.InsertOne(context.Background(), eshuResource)

	return err
}

// CreatePaymentTransactionsResource will store the payment_transaction file details into the database
func (m *MongoService) CreatePaymentTransactionsResource(paymentTransactionsResource *models.PaymentTransactionsResourceDao) error {
	collection := m.db.Collection(m.TransactionsCollection)
	_, err := collection.InsertOne(context.Background(), paymentTransactionsResource)

	return err
}

// CreateRefundResource will store the refund file details into the database
func (m *MongoService) CreateRefundResource(refundResource *models.RefundResourceDao) error {
	collection := m.db.Collection(m.RefundsCollection)
	_, err := collection.InsertOne(context.Background(), refundResource)

	return err
}
