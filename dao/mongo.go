package dao

import (
    "fmt"
    "github.com/companieshouse/payment-reconciliation-consumer/config"
    "github.com/companieshouse/payment-reconciliation-consumer/models"
    "github.com/globalsign/mgo"
)

var session *mgo.Session

// Mongo represents a simplistic MongoDB configuration.
type Mongo struct {
    Config *config.Config
}

// New returns a new Mongo struct using the provided config
func New(cfg *config.Config) *Mongo {

    return &Mongo{
        Config: cfg,
    }
}

// getMongoSession gets a MongoDB Session
func getMongoSession() (*mgo.Session, error) {

    if session == nil {

        var err error

        cfg, err := config.Get()
        if err != nil {
            return nil, fmt.Errorf("error getting config: %s", err)
        }

        session, err = mgo.Dial(cfg.MongoDBURL)
        if err != nil {
            return nil, fmt.Errorf("error dialling into mongodb: %s", err)
        }
    }

    return session.Copy(), nil
}

// CreateEshuResource will store the eshu file details into the database
func (m *Mongo) CreateEshuResource(eshuResource *models.EshuResourceDao) error {

    mongoSession, err := getMongoSession()
    if err != nil {
        return err
    }
    defer mongoSession.Close()

    return mongoSession.DB(m.Config.Database).C(m.Config.ProductsCollection).Insert(eshuResource)
    
}

// CreatePaymentTransactionsResource will store the payment_transaction file details into the database
func (m *Mongo) CreatePaymentTransactionsResource(paymentTransactionsResource *models.PaymentTransactionsResourceDao) error {

    mongoSession, err := getMongoSession()
    if err != nil {
        return err
    }
    defer mongoSession.Close()

    return mongoSession.DB(m.Config.Database).C(m.Config.TransactionsCollection).Insert(paymentTransactionsResource)
}

