package dao

import (
	"github.com/companieshouse/payment-reconciliation-consumer/models"
)

// DAO provides access to the database
type DAO interface {
	CreateEshuResource(dao *models.EshuResourceDao) error
	CreatePaymentTransactionsResource(dao *models.PaymentTransactionsResourceDao) error
}
