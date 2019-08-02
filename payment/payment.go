package payment

import (
	"encoding/json"
	"fmt"
	"github.com/companieshouse/payment-reconciliation-consumer/data"
	"io/ioutil"
	"net/http"

	"github.com/companieshouse/chs.go/log"
)

// ----------------------------------------------------------------------------

// InvalidPaymentAPIResponse is returned when an invalid status is returned
// from the payments api
type InvalidPaymentAPIResponse struct {
	status int
}

func (e *InvalidPaymentAPIResponse) Error() string {
	return fmt.Sprintf("invalid status returned from payments api: [%d]", e.status)
}

// ----------------------------------------------------------------------------

// Get executes a GET request to the specified URL
func Get(paymentAPIURL string, HTTPClient *http.Client, apiKey string) (data.PaymentResponse, error) {
	var p data.PaymentResponse

	req, err := http.NewRequest("GET", paymentAPIURL, nil)
	if err != nil {
		return p, err
	}

	req.SetBasicAuth(apiKey, "")
	log.Trace("GET request to the payment api to get the payment session", log.Data{"Request": paymentAPIURL})

	res, err := HTTPClient.Do(req)
	if err != nil {
		return p, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return p, &InvalidPaymentAPIResponse{res.StatusCode}
	}

	body, err := ioutil.ReadAll(res.Body)
	log.Info("Payment response body", log.Data{"payment": string(body)})
	if err != nil {
		return p, err
	}

	if err := json.Unmarshal(body, &p); err != nil {
		return p, err
	}

	return p, nil
}
