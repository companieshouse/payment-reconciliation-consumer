package dao

import (
	"github.com/companieshouse/payment-reconciliation-consumer/config"
	"github.com/companieshouse/payment-reconciliation-consumer/models"
)

// Service interface declares how to interact with the persistence layer regardless of underlying technology
type Service interface {
	// CreateEshuResource will persist a newly created resource
	CreateEshuResource(dao *models.EshuResourceDao, collectionName string) error
	// CreatePaymentTransactionsResource will persist a newly created resource
	CreatePaymentTransactionsResource(dao *models.PaymentTransactionsResourceDao, collectionName string) error
	// Shutdown can be called to clean up any open resources that the service may be holding on to.
	Shutdown()
}

// NewDAOService will create a new instance of the Service interface. All details about its implementation and the
// database driver will be hidden from outside of this package
func NewDAOService(cfg *config.Config) Service {
	database := getMongoDatabase(cfg.MongoDBURL, cfg.Database)
	return &MongoService{
		db: database,
	}
}
