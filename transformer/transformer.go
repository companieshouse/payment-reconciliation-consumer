package transformer

import (
	"github.com/companieshouse/payment-reconciliation-consumer/config"
	"github.com/companieshouse/payment-reconciliation-consumer/data"
	"github.com/companieshouse/payment-reconciliation-consumer/models"
	"time"
)

// Transformer provides an interface by which to transform payment models to reconciliation entities
type Transformer interface {
	GetEshuResource(payment data.PaymentResponse, paymentDetails data.PaymentDetailsResponse) (models.EshuResourceDao, error)
	GetTransactionResource(payment data.PaymentResponse, paymentDetails data.PaymentDetailsResponse) (models.PaymentTransactionsResourceDao, error)
}

// Transform implements the Transformer interface
type Transform struct{}

// New returns a new implementation of the Transformer interface
func New() *Transform {

	return &Transform{}
}

// GetEshuResource transforms payment data into an Eshu resource entity
func (t *Transform) GetEshuResource(payment data.PaymentResponse, paymentDetails data.PaymentDetailsResponse) (models.EshuResourceDao, error) {

	var eshuResource models.EshuResourceDao

	productMap, err := config.GetProductMap()
	if err != nil {
		return eshuResource, err
	}

	transactionDate, err := time.Parse(time.RFC3339Nano, paymentDetails.TransactionDate)
	if err != nil {
		return eshuResource, err
	}

	eshuResource = models.EshuResourceDao{
		PaymentRef:        paymentDetails.ExternalPaymentID,
		ProductCode:       productMap.Codes[payment.Costs[0].ProductType],
		CompanyNumber:     payment.CompanyNumber,
		FilingDate:        "",
		MadeUpdate:        "",
		TransactionDate:   transactionDate,
	}

	return eshuResource, nil
}

// GetTransactionResource transforms payment data into a payment transaction resource entity
func (t *Transform) GetTransactionResource(payment data.PaymentResponse, paymentDetails data.PaymentDetailsResponse) (models.PaymentTransactionsResourceDao, error) {

	var paymentTransactionsResource models.PaymentTransactionsResourceDao

	transactionDate, err := time.Parse(time.RFC3339Nano, paymentDetails.TransactionDate)
	if err != nil {
		return paymentTransactionsResource, err
	}

	paymentTransactionsResource = models.PaymentTransactionsResourceDao{
		TransactionID:     paymentDetails.ExternalPaymentID,
		TransactionDate:   transactionDate,
		Email:             payment.CreatedBy.Email,
		PaymentMethod:     payment.PaymentMethod,
		Amount:            payment.Amount,
		CompanyNumber:     payment.CompanyNumber,
		TransactionType:   "Immediate bill",
		OrderReference:    payment.Reference,
		Status:            paymentDetails.PaymentStatus,
		UserID:            "system",
		OriginalReference: "",
		DisputeDetails:    ""}

	return paymentTransactionsResource, nil
}
