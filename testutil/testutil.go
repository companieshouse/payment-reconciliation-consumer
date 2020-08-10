package testutil

import (
	"net/http"
	"net/http/httptest"
	"net/url"
)

const CertifiedCopiesOrderGetPaymentSessionResponse = `{
    "amount": "250.00",
    "completed_at": "2020-08-10T07:30:50.495Z",
    "created_at": "2020-08-10T07:28:51.104Z",
    "created_by": {
        "email": "demo@ch.gov.uk",
        "forename": "",
        "id": "67ZeMsvAEgkBWs7tNKacdrPvOmQ",
        "surname": ""
    },
    "description": "",
    "links": {
        "journey": "https://payments.cidev.aws.chdev.org/payments/TqCvKQQSre69nga/pay",
        "resource": "https://api.cidev.aws.chdev.org/basket/checkouts/ORD-971115-970443/payment",
        "self": "payments/TqCvKQQSre69nga"
    },
    "payment_method": "GovPay",
    "reference": "Payments reconciliation testing payment session ref GCI-1312",
    "company_number": "00006400",
    "status": "paid",
    "costs": [
        {
            "amount": "100",
            "available_payment_methods": [
                "credit-card"
            ],
            "class_of_payment": [
                "orderable-item"
            ],
            "description": "certified copy for company 00006400",
            "description_identifier": "certified-copy",
            "product_type": "certified-copy-incorporation-same-day",
            "description_values": {
                "certified-copy": "certified copy for company 00006400",
                "company_number": "00006400"
            }
        },
        {
            "amount": "50",
            "available_payment_methods": [
                "credit-card"
            ],
            "class_of_payment": [
                "orderable-item"
            ],
            "description": "certified copy for company 00006400",
            "description_identifier": "certified-copy",
            "product_type": "certified-copy-same-day",
            "description_values": {
                "certified-copy": "certified copy for company 00006400",
                "company_number": "00006400"
            }
        },
        {
            "amount": "50",
            "available_payment_methods": [
                "credit-card"
            ],
            "class_of_payment": [
                "orderable-item"
            ],
            "description": "certified copy for company 00006400",
            "description_identifier": "certified-copy",
            "product_type": "certified-copy-same-day",
            "description_values": {
                "certified-copy": "certified copy for company 00006400",
                "company_number": "00006400"
            }
        },
        {
            "amount": "50",
            "available_payment_methods": [
                "credit-card"
            ],
            "class_of_payment": [
                "orderable-item"
            ],
            "description": "certified copy for company 00006400",
            "description_identifier": "certified-copy",
            "product_type": "certified-copy-same-day",
            "description_values": {
                "certified-copy": "certified copy for company 00006400",
                "company_number": "00006400"
            }
        }
    ],
    "etag": "15a5ca8f1381b461ad94cd8bee0a165446d54e34faf5ecafc2e27703",
    "kind": "payment-session#payment-session"
}`

func CreateMockClient(hasResponseBody bool, status int, responseBody string) *http.Client {

	mockStreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if hasResponseBody {
			w.Write([]byte(responseBody))
		}
		w.WriteHeader(status)
	}))

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(mockStreamServer.URL)
		},
	}

	httpClient := &http.Client{Transport: transport}

	return httpClient
}
