package transformer

import (
	"github.com/companieshouse/payment-reconciliation-consumer/data"
	_ "github.com/companieshouse/payment-reconciliation-consumer/testing"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

const unparsableTransactionDate = "2020-07-27T09:07:12.864" // Should be "2020-07-27T09:07:12.864Z"

func TestUnitErrorHandling(t *testing.T) {

	Convey("GetEshuResources propagates payment details transaction date parsing error", t, func() {

		// Given
		transformerUnderTest := Transform{}

		// When
		_, err := transformerUnderTest.GetEshuResources(
			data.PaymentResponse{},
			data.PaymentDetailsResponse{TransactionDate: unparsableTransactionDate},
			"paymentId string")

		// Then
		So(err, ShouldNotBeNil)

	})

	Convey("GetTransactionResources propagates payment details transaction date parsing error", t, func() {

		// Given
		transformerUnderTest := Transform{}

		// When
		_, err := transformerUnderTest.GetTransactionResources(
			data.PaymentResponse{},
			data.PaymentDetailsResponse{TransactionDate: unparsableTransactionDate},
			"paymentId string")

		// Then
		So(err, ShouldNotBeNil)

	})
	Convey("GetRefundResource propagates refund date parsing error", t, func() {

		// Given
		transformerUnderTest := Transform{}

		// When
		_, err := transformerUnderTest.GetRefundResource(
			data.PaymentResponse{},
			data.RefundResource{CreatedAt: unparsableTransactionDate},
			"paymentId string")

		// Then
		So(err, ShouldNotBeNil)

	})

}
