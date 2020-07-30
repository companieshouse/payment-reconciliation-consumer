package testutil

// TODO GCI-1032 Refactor existing test code to use this where appropriate.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
)

// TODO GCI-1032 Would a better response, eg with differing costs, be useful?
const CertifiedCopiesOrderGetPaymentSessionResponse = `{
    "amount": "200.00",
    "completed_at": "2020-07-27T09:07:12.864Z",
    "created_at": "2020-07-27T08:55:06.978Z",
    "created_by": {
        "email": "demo@ch.gov.uk",
        "forename": "",
        "id": "67ZeMsvAEgkBWs7tNKacdrPvOmQ",
        "surname": ""
    },
    "description": "",
    "links": {
        "journey": "https://payments.cidev.aws.chdev.org/payments/FIyWA4nsgUMyDak/pay",
        "resource": "https://api.cidev.aws.chdev.org/basket/checkouts/ORD-845315-958394/payment",
        "self": "payments/FIyWA4nsgUMyDak"
    },
    "payment_method": "GovPay",
    "reference": "Payments reconciliation testing payment session ref",
    "company_number": "00006400",
    "status": "paid",
    "costs": [
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
    "etag": "a7b4eff1d3025251a659225dfbe06827a6a259324af14699f2f7884e",
    "kind": "payment-session#payment-session"
}`

func CreateMockClient(hasResponseBody bool, status int, testData string) *http.Client {

	mockStreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if hasResponseBody {
			w.Write([]byte(testData))
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
