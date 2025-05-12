package dao

import (
	"context"
	"testing"

	"github.com/companieshouse/payment-reconciliation-consumer/models"
	testutils "github.com/companieshouse/payment-reconciliation-consumer/testutil"
	. "github.com/smartystreets/goconvey/convey"
)

func TestIntegrationGetMongoClient(t *testing.T) {

	Convey("Given that testcontainers has started a mongo container", t, func() {
		container, uri, _ := testutils.SetupMongoContainer()
		defer container.Terminate(context.Background())

		client := getMongoClient(uri)
		defer client.Disconnect(context.Background())

		Convey("Then getMongoClient should create a valid mongo client instance using the supplied uri", func() {
			So(client, ShouldNotBeNil)
		})

		Convey("Test MongoDatabaseConnection", func() {
			getMongoDatabase(uri, "test")
		})

		Convey("Create Eshu resource will store into the new database", func() {
			eshuResource := &models.EshuResourceDao{
				PaymentRef:    "test-payment-ref",
				ProductCode:   12345,
				CompanyNumber: "test-company-number",
			}
			m := &MongoService{
				db:                     getMongoDatabase(uri, "test"),
				TransactionsCollection: "transactions",
				ProductsCollection:     "products",
				RefundsCollection:      "refunds",
			}
			err := m.CreateEshuResource(eshuResource)
			So(err, ShouldBeNil)
			// Verify that the resource was created

			db := getMongoDatabase(uri, "test")
			So(db.Collection("products").FindOne(context.Background(), map[string]interface{}{"payment_ref": "test-payment-ref"}).Err(), ShouldNotBeNil)

		})

		Convey("Create Transactions resource will store into the new database", func() {
			paymentTransactionsResource := &models.PaymentTransactionsResourceDao{
				TransactionID: "test-transactrion-id",
				Email:         "test-email",
				CompanyNumber: "test-company-number",
			}
			m := &MongoService{
				db:                     getMongoDatabase(uri, "test"),
				TransactionsCollection: "transactions",
				ProductsCollection:     "products",
				RefundsCollection:      "refunds",
			}
			err := m.CreatePaymentTransactionsResource(paymentTransactionsResource)
			So(err, ShouldBeNil)
			// Verify that the resource was created

			db := getMongoDatabase(uri, "test")
			So(db.Collection("transactions").FindOne(context.Background(), map[string]interface{}{"transaction_id": "test-transactrion-id"}).Err(), ShouldBeNil)
		})

		Convey("Create Refund resource will store into the new database", func() {
			refundResource := &models.RefundResourceDao{
				TransactionID: "test-transactrion-id",
				Email:         "test-email",
				CompanyNumber: "test-company-number",
			}
			m := &MongoService{
				db:                     getMongoDatabase(uri, "test"),
				TransactionsCollection: "transactions",
				ProductsCollection:     "products",
				RefundsCollection:      "refunds",
			}
			err := m.CreateRefundResource(refundResource)
			So(err, ShouldBeNil)
			// Verify that the resource was created

			db := getMongoDatabase(uri, "test")
			So(db.Collection("refunds").FindOne(context.Background(), map[string]interface{}{"transaction_id": "test-transactrion-id"}).Err(), ShouldBeNil)
		})

	})

}
