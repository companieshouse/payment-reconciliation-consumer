package payment

import (
	"encoding/json"
	"fmt"
	"github.com/companieshouse/payment-reconciliation-consumer/data"
	"github.com/companieshouse/payment-reconciliation-consumer/keys"
	"io/ioutil"
	"net/http"

	"github.com/companieshouse/chs.go/log"
)

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
