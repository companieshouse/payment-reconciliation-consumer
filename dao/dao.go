package dao

import (
    "github.com/companieshouse/payment-reconciliation-consumer/models"
)

// Service interface declares how to interact with the persistence layer regardless of underlying technology
type DAO interface {
    // CreateEshuResource will persist a newly created resource
    CreateEshuResource(dao *models.EshuResourceDao) error
    // CreatePaymentTransactionsResource will persist a newly created resource
    CreatePaymentTransactionsResource(dao *models.PaymentTransactionsResourceDao) error

}

