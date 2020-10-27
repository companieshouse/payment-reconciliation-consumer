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

	Convey("GetRefundResource correctly maps data", t, func() {

		// Given
		transformerUnderTest := Transform{}
		paymentId := "paymentId string"

		paymentResponse := data.PaymentResponse{
			CreatedBy: data.Created{
				Email: "exampleEmail",
			},
			PaymentMethod: "credit-card",
			CompanyNumber: "companyNumber",
			Reference:     "example_reference",
			Costs: []data.Cost{{
				ProductType: "ds01",
			}},
		}
		refundResource := data.RefundResource{
			RefundId:  "refundId",
			CreatedAt: "2020-10-21T15:48:30.551Z",
			Amount:    800,
			Status:    "success",
		}

		// When
		resourceDao, err := transformerUnderTest.GetRefundResource(
			paymentResponse,
			refundResource,
			paymentId)

		// Then

		So(err, ShouldBeNil)
		So(resourceDao.Status, ShouldEqual, refundResource.Status)
		So(resourceDao.TransactionID, ShouldEqual, "X"+refundResource.RefundId)
		So(resourceDao.TransactionType, ShouldEqual, "Refund")
		So(resourceDao.TransactionDate, ShouldNotBeNil)
		So(resourceDao.Amount, ShouldEqual, "800")
		So(resourceDao.Email, ShouldEqual, paymentResponse.CreatedBy.Email)
		So(resourceDao.CompanyNumber, ShouldEqual, paymentResponse.CompanyNumber)
		So(resourceDao.PaymentMethod, ShouldEqual, paymentResponse.PaymentMethod)
		So(resourceDao.OrderReference, ShouldEqual, "example-reference")
		So(resourceDao.UserID, ShouldEqual, "system")
		So(resourceDao.OriginalReference, ShouldEqual, "X"+paymentId)
		So(resourceDao.ProductCode, ShouldEqual, 16032)
		So(resourceDao.DisputeDetails, ShouldEqual, "")
	})
}
