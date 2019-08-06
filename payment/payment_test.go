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
    Convey("test successful get request of payment ", t, func() {
        b, statusCode, err := GetPayment("http://test-url.com", createMockClient(true, 200, paymentTestData), "")
        So(err, ShouldBeNil)
        So(statusCode, ShouldEqual, 200)
        So(b, ShouldNotBeEmpty)
    })
    Convey("test error returned when client throws error", t, func() {
        _, statusCode, err := GetPayment("test-url.com", createMockClient(false, 500, paymentTestData), "")
        So(err, ShouldNotBeNil)
        So(statusCode, ShouldEqual, 500)
    })
    Convey("test error returned when invalid http status returned", t, func() {
        _, statusCode, err := GetPayment("http://test-url.com", createMockClient(false, 404, paymentTestData), "")
        So(err, ShouldNotBeNil)
        So(statusCode, ShouldEqual, 404)
    })

}
func TestUnitGetDetailsPayment(t *testing.T) {
    Convey("test successful get request of payment ", t, func() {
        b, statusCode, err := GetPaymentDetails("http://test-url.com", createMockClient(true, 200,paymentDetailsTestData), "")
        So(err, ShouldBeNil)
        So(statusCode, ShouldEqual, 200)
        So(b, ShouldNotBeEmpty)
    })
    Convey("test error returned when client throws error", t, func() {
        _, statusCode, err := GetPaymentDetails("test-url.com", createMockClient(false, 500,paymentDetailsTestData), "")
        So(err, ShouldNotBeNil)
        So(statusCode, ShouldEqual, 500)
    })
    Convey("test error returned when invalid http status returned", t, func() {
        _, statusCode, err := GetPaymentDetails("http://test-url.com", createMockClient(false, 404,paymentDetailsTestData), "")
        So(err, ShouldNotBeNil)
        So(statusCode, ShouldEqual, 404)
    })

}

