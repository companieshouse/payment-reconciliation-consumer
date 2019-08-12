package transformer

import (
	"github.com/companieshouse/payment-reconciliation-consumer/config"
	"github.com/companieshouse/payment-reconciliation-consumer/data"
	"github.com/companieshouse/payment-reconciliation-consumer/models"
)

// Transformer provides an interface by which to transform payment models to reconciliation entities
type Transformer interface {
	GetEshuResource(payment data.PaymentResponse, paymentDetails data.PaymentDetailsResponse) (models.EshuResourceDao, error)
	GetTransactionResource(payment data.PaymentResponse, paymentDetails data.PaymentDetailsResponse) models.PaymentTransactionsResourceDao
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

	eshuResource = models.EshuResourceDao{
		PaymentRef:    paymentDetails.PaymentID,
		ProductCode:   productMap.Codes[payment.Costs[0].ProductType],
		CompanyNumber: payment.CompanyNumber,
		FilingDate:    "",
		MadeUpdate:    "",
		TransactionDate:   paymentDetails.TransactionDate,
	}

	return eshuResource, nil
}

// GetTransactionResource transforms payment data into a payment transaction resource entity
func (t *Transform) GetTransactionResource(payment data.PaymentResponse, paymentDetails data.PaymentDetailsResponse) models.PaymentTransactionsResourceDao {

	return models.PaymentTransactionsResourceDao{
		TransactionID:     paymentDetails.PaymentID,
		TransactionDate:   paymentDetails.TransactionDate,
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
}
