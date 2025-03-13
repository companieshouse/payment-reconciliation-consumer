package dao

import (
	"github.com/companieshouse/payment-reconciliation-consumer/config"
	"github.com/companieshouse/payment-reconciliation-consumer/models"
)

// DAO provides access to the database
type DAO interface {
	CreateEshuResource(dao *models.EshuResourceDao) error
	CreatePaymentTransactionsResource(dao *models.PaymentTransactionsResourceDao) error
	CreateRefundResource(dao *models.RefundResourceDao) error
}

func NewPaymentReconciliationDAOService(cfg *config.Config) DAO {
	database := getMongoDatabase(cfg.MongoDBURL, cfg.Database)
	return &MongoService{
		db:                     database,
		TransactionsCollection: cfg.TransactionsCollection,
		ProductsCollection:     cfg.ProductsCollection,
		RefundsCollection:      cfg.RefundsCollection,
	}
}
