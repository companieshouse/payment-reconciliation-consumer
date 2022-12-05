package transformer

import (
	"github.com/companieshouse/payment-reconciliation-consumer/config"
	"github.com/companieshouse/payment-reconciliation-consumer/data"
	"github.com/companieshouse/payment-reconciliation-consumer/models"
	"strconv"
	"strings"
	"time"
)

// Transformer provides an interface by which to transform payment models to reconciliation entities
type Transformer interface {
	GetEshuResources(payment data.PaymentResponse, paymentDetails data.PaymentDetailsResponse, paymentId string) ([]models.EshuResourceDao, error)
	GetTransactionResources(payment data.PaymentResponse, paymentDetails data.PaymentDetailsResponse, paymentId string) ([]models.PaymentTransactionsResourceDao, error)
	GetRefundResource(payment data.PaymentResponse, refund data.RefundResource, paymentId string) (models.RefundResourceDao, error)
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

// GetTransactionResources transforms payment data into payment transaction resource entities
func (t *Transform) GetRefundResource(payment data.PaymentResponse,
	refund data.RefundResource,
	paymentId string) (models.RefundResourceDao, error) {

	refundResource := models.RefundResourceDao{}

	refundDate, err := time.Parse(time.RFC3339Nano, refund.CreatedAt)
	if err != nil {
		return refundResource, err
	}

	productMap, err := config.GetProductMap()
	if err != nil {
		return refundResource, err
	}

	refundResource = models.RefundResourceDao{
		TransactionID:     "X" + refund.RefundId,
		TransactionDate:   refundDate,
		RefundID:          refund.RefundId,
		RefundedAt:        refund.RefundedAt,
		PaymentID:         paymentId,
		Email:             payment.CreatedBy.Email,
		PaymentMethod:     payment.PaymentMethod,
		Amount:            strconv.Itoa(refund.Amount / 100),
		CompanyNumber:     payment.CompanyNumber,
		TransactionType:   "Refund",
		OrderReference:    strings.Replace(payment.Reference, "_", "-", -1),
		Status:            refund.Status,
		UserID:            "system",
		OriginalReference: "X" + paymentId,
		DisputeDetails:    "",
		ProductCode:       productMap.Codes[payment.Costs[0].ProductType],
	}

	return refundResource, nil
}
