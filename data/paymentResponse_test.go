package data

import (
    . "github.com/smartystreets/goconvey/convey"
    "testing"
)

func TestUnitGetCosts(t *testing.T) {
    Convey("Get Costs - valid", t, func() {
        costs := []Cost{{DescriptionIdentifier:"abc"},{DescriptionIdentifier:"cic-report",Amount:"15"}}
        paymentResponse := PaymentResponse{Costs:costs}
        cost, err := paymentResponse.GetCost("cic-report")
        So(err, ShouldBeNil)
        So(cost.DescriptionIdentifier,ShouldEqual,"cic-report")
        So(cost.Amount,ShouldEqual,"15")
    })
    Convey("Get Costs - Invalid", t, func() {
        costs := []Cost{{DescriptionIdentifier:"abc"},{DescriptionIdentifier:"cic",Amount:"15"}}
        paymentResponse := PaymentResponse{Costs:costs}
        cost, err := paymentResponse.GetCost("cic-report")
        So(err, ShouldNotBeNil)
        So(cost.DescriptionIdentifier, ShouldBeBlank)
    })



}

