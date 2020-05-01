package data

import (
	. "github.com/smartystreets/goconvey/convey"
	. "github.com/stretchr/testify/assert"
	"testing"
)

func TestUnitIsReconcilable(t *testing.T) {

	Convey("data-maintenance payments are reconcilable", t, func() {
		dataMaintenance := createPaymentResponse(DataMaintenance)
		Equal(t, dataMaintenance.IsReconcilable(), true, "data-maintenance payments should be reconcilable")
	})

	Convey("orderable-item payments are reconcilable", t, func() {
		orderableItem := createPaymentResponse(OrderableItem)
		Equal(t, orderableItem.IsReconcilable(), true, "orderable-item payments should be reconcilable")
	})

	Convey("penalty payments are not reconcilable", t, func() {
		penalty := createPaymentResponse(Penalty)
		Equal(t, penalty.IsReconcilable(), false, "penalty payments should not be reconcilable")
	})

}

// Creates a payment response with the class of payment specified.
func createPaymentResponse(classOfPayment string) PaymentResponse {
	cost := Cost{
		ClassOfPayment: []string{classOfPayment},
	}
	paymentResponse := PaymentResponse{
		Costs: []Cost{cost},
	}
	return paymentResponse
}
