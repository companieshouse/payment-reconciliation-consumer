package payment

import (
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

const paymentTestData = `{
    "amount": "15.00",
    "completed_at": "2019-08-05T13:04:34.695Z",
    "created_at": "2019-08-05T13:04:05.737Z",
    "created_by": {
        "email": "test@companieshouse.gov.uk",
        "forename": "",
        "id": "Y2VkZWVlMzhlZWFjY2M4MzQ3JB",
        "surname": ""
    },
    "description": "Small Full Accounts",
    "links": {
        "journey": "https://payments_url/payments/98758411565010245737/pay",
        "resource": "https://transaction_url/transactions/188389-321115-650101/payment",
        "self": "payments/98758411565010245737"
    },
    "payment_method": "GovPay",
    "reference": "cic_report_and_accounts_188389-321115-650101",
    "company_number": "00000000",
    "status": "paid",
    "costs": [
        {
            "amount": "15",
            "available_payment_methods": [
                "credit-card",
                "account"
            ],
            "class_of_payment": [
                "data-maintenance"
            ],
            "description": "CIC report and accounts",
            "description_identifier": "cic-report",
            "product_type": "cic-report",
            "description_values": null
        }
    ],
    "etag": "728ee7a32eea8fe0e65515705e1dafc447c39703b62505df0be2f241",
    "kind": "payment-session#payment-session"
}`

const paymentDetailsTestData = `{
    "card_type": "Visa",
    "payment_id": "lp9o81j5pgo0efsscq86vsn7pn",
    "transaction_date": "2019-07-12T12:26:32.786Z",
    "payment_status": "accepted"
}`

// TODO GCI-1032 Would a better response, eg with differing costs, be useful?
const certifiedCopiesOrderGetPaymentSessionResponse = `{
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

func createMockClient(hasResponseBody bool, status int, testData string) *http.Client {

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

func TestUnitGetPayment(t *testing.T) {

	p := Fetch{}

	Convey("test successful get request of payment ", t, func() {
		b, statusCode, err := p.GetPayment("http://test-url.com", createMockClient(true, 200, paymentTestData), "")
		So(err, ShouldBeNil)
		So(statusCode, ShouldEqual, 200)
		So(b, ShouldNotBeEmpty)
	})

	Convey("test error returned when client throws error", t, func() {
		_, statusCode, err := p.GetPayment("test-url.com", createMockClient(false, 500, paymentTestData), "")
		So(err, ShouldNotBeNil)
		So(statusCode, ShouldEqual, 500)
	})

	Convey("test error returned when invalid http status returned", t, func() {
		_, statusCode, err := p.GetPayment("http://test-url.com", createMockClient(false, 404, paymentTestData), "")
		So(err, ShouldNotBeNil)
		So(statusCode, ShouldEqual, 404)
	})

	Convey("test successful get request for certified copies order payment session contains expected costs", t, func() {
		b, statusCode, err := p.GetPayment("http://test-url.com", createMockClient(true, 200, certifiedCopiesOrderGetPaymentSessionResponse), "")
		So(err, ShouldBeNil)
		So(statusCode, ShouldEqual, 200)
		So(b, ShouldNotBeEmpty)
		So(len(b.Costs), ShouldEqual, 4)
		for _, cost := range b.Costs {
			So(cost.Amount, ShouldEqual, `50`)
		}
	})
}

func TestUnitGetDetailsPayment(t *testing.T) {

	p := Fetch{}

	Convey("test successful get request of payment ", t, func() {
		b, statusCode, err := p.GetPaymentDetails("http://test-url.com", createMockClient(true, 200, paymentDetailsTestData), "")
		So(err, ShouldBeNil)
		So(statusCode, ShouldEqual, 200)
		So(b, ShouldNotBeEmpty)
	})

	Convey("test error returned when client throws error", t, func() {
		_, statusCode, err := p.GetPaymentDetails("test-url.com", createMockClient(false, 500, paymentDetailsTestData), "")
		So(err, ShouldNotBeNil)
		So(statusCode, ShouldEqual, 500)
	})

	Convey("test error returned when invalid http status returned", t, func() {
		_, statusCode, err := p.GetPaymentDetails("http://test-url.com", createMockClient(false, 404, paymentDetailsTestData), "")
		So(err, ShouldNotBeNil)
		So(statusCode, ShouldEqual, 404)
	})
}
