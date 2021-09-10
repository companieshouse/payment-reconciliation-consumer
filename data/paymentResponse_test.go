package data

import (
	. "github.com/smartystreets/goconvey/convey"
	. "github.com/stretchr/testify/assert"
	"testing"
	"github.com/companieshouse/payment-reconciliation-consumer/config"
	"github.com/companieshouse/chs.go/log"
	"fmt"
	"path/filepath"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

func TestUnitIsReconcilable(t *testing.T) {

	productMap, err := createProductMap()

	if err != nil {
		log.Error(fmt.Errorf("error initialising productMap: %s", err), nil)
	}

	Convey("data-maintenance payments are reconcilable", t, func() {
		dataMaintenance := createPaymentResponse(DataMaintenance, "ds01")
		Equal(t, dataMaintenance.IsReconcilable(productMap), true, "data-maintenance payments should be reconcilable")
	})

	Convey("orderable-item payments are reconcilable", t, func() {
		orderableItem := createPaymentResponse(OrderableItem, "certificate")
		Equal(t, orderableItem.IsReconcilable(productMap), true, "orderable-item payments should be reconcilable")
	})

	Convey("penalty payments are not reconcilable", t, func() {
		penalty := createPaymentResponse(Penalty, "lfp")
		Equal(t, penalty.IsReconcilable(productMap), false, "penalty payments should not be reconcilable")
	})

	Convey("penalty payments are not reconcilable", t, func() {
		penalty := createPaymentResponse(Legacy, "webfiling")
		Equal(t, penalty.IsReconcilable(productMap), false, "legacy payments should not be reconcilable")
	})

	Convey("penalty payments are not reconcilable", t, func() {
		penalty := createPaymentResponse(DataMaintenance, "extractives")
		Equal(t, penalty.IsReconcilable(productMap), false, "extractives payments should not be reconcilable")
	})

}

// Creates a payment response with the class of payment specified.
func createPaymentResponse(classOfPayment string, productType string) PaymentResponse {
	cost := Cost{
		ClassOfPayment: []string{classOfPayment},
		ProductType: productType,
	}
	paymentResponse := PaymentResponse{
		Costs: []Cost{cost},
	}
	return paymentResponse
}

func createProductMap() (*config.ProductMap, error) {
	var productMap *config.ProductMap

	filename, err := filepath.Abs("../assets/product_code.yml")
	if err != nil {
		return nil, err
	}

	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yamlFile, &productMap)
	if err != nil {
		return nil, err
	}

	log.Info("Product map config has been loaded", log.Data{"products": productMap})

	return productMap, nil
}
