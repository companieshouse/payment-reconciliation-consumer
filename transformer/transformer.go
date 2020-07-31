package transformer

import (
	"github.com/companieshouse/payment-reconciliation-consumer/config"
	"github.com/companieshouse/payment-reconciliation-consumer/data"
	"github.com/companieshouse/payment-reconciliation-consumer/models"
	"strings"
	"time"
)

// Transformer provides an interface by which to transform payment models to reconciliation entities
type Transformer interface {
	GetEshuResources(payment data.PaymentResponse, paymentDetails data.PaymentDetailsResponse, paymentId string) ([]models.EshuResourceDao, error)
	GetTransactionResources(payment data.PaymentResponse, paymentDetails data.PaymentDetailsResponse, paymentId string) ([]models.PaymentTransactionsResourceDao, error)
}

// Transform implements the Transformer interface
type Transform struct{}

// New returns a new implementation of the Transformer interface
func New() *Transform {

	return &Transform{}
}

// GetEshuResources transforms payment data into Eshu resource entities
func (t *Transform) GetEshuResources(payment data.PaymentResponse,
	paymentDetails data.PaymentDetailsResponse,
	paymentId string) ([]models.EshuResourceDao, error) {

	eshuResources := []models.EshuResourceDao{}

	productMap, err := config.GetProductMap()
	if err != nil {
		return eshuResources, err
	}

	transactionDate, err := time.Parse(time.RFC3339Nano, paymentDetails.TransactionDate)
	if err != nil {
		return eshuResources, err
	}

	for _, cost := range payment.Costs {
		eshuResources = append(eshuResources, models.EshuResourceDao{
			PaymentRef:      "X" + paymentId,
			ProductCode:     productMap.Codes[cost.ProductType],
			CompanyNumber:   payment.CompanyNumber,
			FilingDate:      "",
			MadeUpdate:      "",
			TransactionDate: transactionDate,
		})
	}

	return eshuResources, nil
}

// GetTransactionResources transforms payment data into payment transaction resource entities
func (t *Transform) GetTransactionResources(payment data.PaymentResponse,
	paymentDetails data.PaymentDetailsResponse,
	paymentId string) ([]models.PaymentTransactionsResourceDao, error) {

	paymentTransactionsResources := []models.PaymentTransactionsResourceDao{}

	transactionDate, err := time.Parse(time.RFC3339Nano, paymentDetails.TransactionDate)
	if err != nil {
		return paymentTransactionsResources, err
	}

	// TODO GCI-1032 So how is Finance reconciliation going to tally eshus (products) with transactions?
	for _, cost := range payment.Costs {
		paymentTransactionsResources = append(paymentTransactionsResources, models.PaymentTransactionsResourceDao{
			TransactionID:     "X" + paymentId,
			TransactionDate:   transactionDate,
			Email:             payment.CreatedBy.Email,
			PaymentMethod:     payment.PaymentMethod,
			Amount:            cost.Amount,
			CompanyNumber:     payment.CompanyNumber,
			TransactionType:   "Immediate bill",
			OrderReference:    strings.Replace(payment.Reference, "_", "-", -1),
			Status:            paymentDetails.PaymentStatus,
			UserID:            "system",
			OriginalReference: "",
			DisputeDetails:    "",
		})
	}

	return paymentTransactionsResources, nil
}
