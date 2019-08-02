package dao

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payment-reconciliation-consumer/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

func getMongoClient(mongoDBURL string) *mongo.Client {
	if client != nil {
		return client
	}

	ctx := context.Background()

	clientOptions := options.Client().ApplyURI(mongoDBURL)
	client, err := mongo.Connect(ctx, clientOptions)

	// assume the caller of this func cannot handle the case where there is no database connection so the prog must
	// crash here as the service cannot continue.
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// check we can connect to the mongodb instance. failure here should result in a crash.
	pingContext, cancel := context.WithDeadline(ctx, time.Now().Add(5*time.Second))
	defer cancel()
	err = client.Ping(pingContext, nil)
	if err != nil {
		log.Error(errors.New("ping to mongodb timed out. please check the connection to mongodb and that it is running"))
		os.Exit(1)
	}

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
	db MongoDatabaseInterface
}

// EshuResource will store the payable request into the database
func (m *MongoService) CreateEshuResource(dao *models.EshuResourceDao, collectionName string) error {

	dao.ID = primitive.NewObjectID()

	collection := m.db.Collection(collectionName)
	_, err := collection.InsertOne(context.Background(), dao)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// EshuResource will store the payable request into the database
func (m *MongoService) CreatePaymentTransactionsResource(dao *models.PaymentTransactionsResourceDao, collectionName string) error {

	dao.ID = primitive.NewObjectID()

	collection := m.db.Collection(collectionName)
	_, err := collection.InsertOne(context.Background(), dao)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// Shutdown is a hook that can be used to clean up db resources
func (m *MongoService) Shutdown() {
	if client != nil {
		err := client.Disconnect(context.Background())
		if err != nil {
			log.Error(err)
			return
		}
		log.Info("disconnected from mongodb successfully")
	}
}
