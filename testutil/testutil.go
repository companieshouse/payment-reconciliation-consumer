package testutil

import (
	"net/http"
	"net/http/httptest"
	"net/url"
)

const CertifiedCopiesOrderGetPaymentSessionResponse = `{
    "amount": "250.00",
    "completed_at": "2020-08-04T07:37:56.432Z",
    "created_at": "2020-08-04T07:34:34.821Z",
    "created_by": {
        "email": "demo@ch.gov.uk",
        "forename": "",
        "id": "67ZeMsvAEgkBWs7tNKacdrPvOmQ",
        "surname": ""
    },
    "description": "",
    "links": {
        "journey": "https://payments.cidev.aws.chdev.org/payments/eXsiQna15i1tfpE/pay",
        "resource": "https://api.cidev.aws.chdev.org/basket/checkouts/ORD-366015-965264/payment",
        "self": "payments/eXsiQna15i1tfpE"
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
    "etag": "b8a97d0bd63b590b7f1c90eb22778f55eea627027656145a678ae4e1",
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
