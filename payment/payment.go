package payment

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/companieshouse/payment-reconciliation-consumer/data"
	"github.com/companieshouse/payment-reconciliation-consumer/keys"
	"io/ioutil"
	"net/http"

	"github.com/companieshouse/chs.go/log"
)

// ErrResourceGone is a sentinel error used when payment resources have been intentionally removed
var ErrResourceGone = errors.New(`The requested resource for payment is gone - status [410]`)

// InvalidPaymentAPIResponse is returned when an invalid status is returned from the payments api
type InvalidPaymentAPIResponse struct {
	status int
}

// Error provides a consistent error when receiving an invalid response status when fetching payments
func (e *InvalidPaymentAPIResponse) Error() string {
	return fmt.Sprintf("invalid status returned from payments api: [%d]", e.status)
}

// Fetcher provides an interface by which to fetch payments data
type Fetcher interface {
	GetPayment(paymentAPIURL string, HTTPClient *http.Client, apiKey string) (data.PaymentResponse, int, error)
	GetPaymentDetails(paymentAPIURL string, HTTPClient *http.Client, apiKey string) (data.PaymentDetailsResponse, int, error)
	GetLatestRefundStatus(paymentAPIURL string, HTTPClient *http.Client, apiKey string) (*data.RefundResource, int, error)
}

// Fetch implements the the Fetcher interface
type Fetch struct{}

// New returns a new implementation of the Fetcher interface
func New() *Fetch {

	return &Fetch{}
}

// GetPayment executes a GET request to payment URL
func (impl *Fetch) GetPayment(paymentAPIURL string, HTTPClient *http.Client, apiKey string) (data.PaymentResponse, int, error) {
	var p data.PaymentResponse

	req, err := http.NewRequest("GET", paymentAPIURL, nil)
	if err != nil {
		return p, 0, err
	}

	req.SetBasicAuth(apiKey, "")
	log.Trace("GET request to the payment api to get the payment session", log.Data{keys.Request: paymentAPIURL})

	res, err := HTTPClient.Do(req)
	if err != nil {
		return p, 500, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusGone {
			return p, res.StatusCode, ErrResourceGone
		}
		return p, res.StatusCode, &InvalidPaymentAPIResponse{res.StatusCode}
	}

	body, err := ioutil.ReadAll(res.Body)
	log.Info("Payment response body", log.Data{keys.Payment: string(body)})
	if err != nil {
		return p, res.StatusCode, err
	}

	if err := json.Unmarshal(body, &p); err != nil {
		return p, res.StatusCode, err
	}

	return p, res.StatusCode, nil
}

// GetPaymentDetails executes a GET request to payment Details URL
func (impl *Fetch) GetPaymentDetails(paymentAPIURL string, HTTPClient *http.Client, apiKey string) (data.PaymentDetailsResponse, int, error) {
	var p data.PaymentDetailsResponse

	req, err := http.NewRequest("GET", paymentAPIURL, nil)
	if err != nil {
		return p, 0, err
	}

	req.SetBasicAuth(apiKey, "")
	log.Trace("GET request to the payment api to get the payment details", log.Data{keys.Request: paymentAPIURL})

	res, err := HTTPClient.Do(req)
	if err != nil {
		return p, 500, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return p, res.StatusCode, &InvalidPaymentAPIResponse{res.StatusCode}
	}

	body, err := ioutil.ReadAll(res.Body)
	log.Info("Payment details response body", log.Data{keys.PaymentDetails: string(body)})
	if err != nil {
		return p, res.StatusCode, err
	}

	if err := json.Unmarshal(body, &p); err != nil {
		return p, res.StatusCode, err
	}

	return p, res.StatusCode, nil
}

func (impl *Fetch) GetLatestRefundStatus(refundEndpointUrl string, HTTPClient *http.Client, apiKey string) (*data.RefundResource, int, error) {
	var p data.RefundResource

	req, err := http.NewRequest("PATCH", refundEndpointUrl, nil)
	if err != nil {
		return &p, 0, err
	}

	req.SetBasicAuth(apiKey, "")
	log.Trace("PATCH request to the payment api to update and fetch latest refund information", log.Data{keys.Request: refundEndpointUrl})

	res, err := HTTPClient.Do(req)
	if err != nil {
		return &p, 500, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return &p, res.StatusCode, &InvalidPaymentAPIResponse{res.StatusCode}
	}

	body, err := ioutil.ReadAll(res.Body)
	log.Info("Refund response body", log.Data{keys.RefundDetails: string(body)})
	if err != nil {
		return &p, res.StatusCode, err
	}

	if err := json.Unmarshal(body, &p); err != nil {
		return &p, res.StatusCode, err
	}

	return &p, res.StatusCode, nil
}
