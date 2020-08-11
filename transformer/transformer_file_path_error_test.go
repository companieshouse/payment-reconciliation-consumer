package transformer

import (
	"github.com/companieshouse/payment-reconciliation-consumer/data"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestUnitFilePathErrorHandling(t *testing.T) {

	// This test happens to work correctly because the path defaults to the current directory.
	Convey("GetEshuResources propagates product_codes.yml file path  error", t, func() {

		// Given
		transformerUnderTest := Transform{}

		// When
		_, err := transformerUnderTest.GetEshuResources(
			data.PaymentResponse{},
			data.PaymentDetailsResponse{},
			"paymentId string")

		// Then
		So(err, ShouldNotBeNil)

	})

}
