package dao

import (
    "fmt"
    "github.com/companieshouse/payment-reconciliation-consumer/models"
    "github.com/companieshouse/payment-reconciliation-consumer/config"
    "github.com/globalsign/mgo"
)

var session *mgo.Session

// Mongo represents a simplistic MongoDB configuration.
type Mongo struct {
    Config *config.Config
}

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

// EshuResource will store the payable request into the database
func (m *Mongo) CreateEshuResource(eshuResource *models.EshuResourceDao) error {


    eshuSession, err := getMongoSession()
    if err != nil {
        return err
    }
    defer eshuSession.Close()

    cfg, err := config.Get()
    if err != nil {
        return fmt.Errorf("error getting config: %s", err)
    }
    c := eshuSession.DB(cfg.Database).C(cfg.TransactionsCollection)

    return c.Insert(eshuResource)
    
}

// PaymentTransactionResource will store the payable request into the database
func (m *Mongo) CreatePaymentTransactionsResource(paymentTransactionsResource *models.PaymentTransactionsResourceDao) error {
    paymentTransactionsSession, err := getMongoSession()
    if err != nil {
        return err
    }
    defer paymentTransactionsSession.Close()

    cfg, err := config.Get()
    if err != nil {
        return fmt.Errorf("error getting config: %s", err)
    }
    c := paymentTransactionsSession.DB(cfg.Database).C(cfg.ProductsCollection)

    return c.Insert(paymentTransactionsResource)
   
}

