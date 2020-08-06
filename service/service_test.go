package service

import (
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/companieshouse/chs.go/kafka/consumer/cluster"
	"github.com/companieshouse/chs.go/kafka/producer"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payment-reconciliation-consumer/config"
	"github.com/companieshouse/payment-reconciliation-consumer/dao"
	"github.com/companieshouse/payment-reconciliation-consumer/data"
	"github.com/companieshouse/payment-reconciliation-consumer/models"
	"github.com/companieshouse/payment-reconciliation-consumer/payment"
	_ "github.com/companieshouse/payment-reconciliation-consumer/testing"
	"github.com/companieshouse/payment-reconciliation-consumer/testutil"
	"github.com/companieshouse/payment-reconciliation-consumer/transformer"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const paymentsAPIUrl = "paymentsAPIUrl"
const apiKey = "apiKey"
const paymentResourceID = "paymentResourceID"
const waitTimeoutSeconds = 1
const timeout = waitTimeoutSeconds * time.Second

func createMockService(productMap *config.ProductMap, mockPayment *payment.MockFetcher, mockTransformer *transformer.MockTransformer, mockDao *dao.MockDAO) *Service {

	return &Service{
		Producer:       createMockProducer(),
		PpSchema:       getDefaultSchema(),
		Payments:       mockPayment,
		Transformer:    mockTransformer,
		DAO:            mockDao,
		PaymentsAPIURL: paymentsAPIUrl,
		APIKey:         apiKey,
		ProductMap:     productMap,
		Client:         &http.Client{},
		StopAtOffset:   int64(-1),
		Topic:          "test",
	}
}

// Creates a somewhat mocked out Service instance that importantly uses a real Transformer (Transform)
// allowing a form of integration testing as far as the service and transformer are concerned.
func createMockServiceWithRealTransformer(productMap *config.ProductMap,
	mockPayment *payment.MockFetcher,
	mockDao *dao.MockDAO) *Service {

	return &Service{
		Producer:       createMockProducer(),
		PpSchema:       getDefaultSchema(),
		Payments:       mockPayment,
		Transformer:    transformer.New(),
		DAO:            mockDao,
		PaymentsAPIURL: paymentsAPIUrl,
		APIKey:         apiKey,
		ProductMap:     productMap,
		Client:         &http.Client{},
		StopAtOffset:   int64(-1),
		Topic:          "test",
	}
}

func createMockConsumerWithMessage() *consumer.GroupConsumer {

	return &consumer.GroupConsumer{
		GConsumer: MockConsumer{},
		Group:     MockGroup{},
	}
}

func createMockProducer() *producer.Producer {

	return &producer.Producer{
		SyncProducer: MockProducer{},
	}
}

func createProductMap() (*config.ProductMap, error) {
	var productMap *config.ProductMap

	filename, err := filepath.Abs("assets/product_code.yml")
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

func getDefaultSchema() string {

	return "{\"type\":\"record\",\"name\":\"payment_processed\",\"namespace\":\"payments\",\"fields\":[{\"name\":\"payment_resource_id\",\"type\":\"string\"}]}"
}

type MockProducer struct {
	sarama.SyncProducer
}

func (m MockProducer) Close() error {
	return nil
}

type MockConsumer struct{}

func (m MockConsumer) Close() error {
	return nil
}

func (m MockConsumer) Messages() <-chan *sarama.ConsumerMessage {
	out := make(chan *sarama.ConsumerMessage)

	go func() {
		out <- &sarama.ConsumerMessage{
			Value: []byte("\"" + paymentResourceID + "\""),
		}
		close(out)
	}()

	return out
}

func (m MockConsumer) Errors() <-chan error {
	return nil
}

type MockGroup struct{}

func (m MockGroup) MarkOffset(msg *sarama.ConsumerMessage, metadata string) {

}

func (m MockGroup) CommitOffsets() error {
	return nil
}

func TestUnitStart(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	productMap, err := createProductMap()
	if err != nil {
		log.Error(fmt.Errorf("error initialising productMap: %s", err), nil)
	}

	Convey("Successful process of a single Kafka message for a 'data-maintenance' payment", t, func() {
		processingOfPaymentKafkaMessageCreatesReconciliationRecords(ctrl, productMap, data.DataMaintenance)
	})

	Convey("Process of a single Kafka message for a 'penalty' payment", t, func() {

		wg := &sync.WaitGroup{}
		wg.Add(1)
		c := make(chan os.Signal)

		mockPayment := payment.NewMockFetcher(ctrl)
		mockTransformer := transformer.NewMockTransformer(ctrl)
		mockDao := dao.NewMockDAO(ctrl)

		svc := createMockService(productMap, mockPayment, mockTransformer, mockDao)

		Convey("Given a message is readily available for the service to consume", func() {
			svc.Consumer = createMockConsumerWithMessage()

			Convey("When the payment corresponding to the message is fetched successfully", func() {

				cost := data.Cost{
					ClassOfPayment: []string{data.Penalty},
				}

				pr := data.PaymentResponse{
					Costs: []data.Cost{cost},
				}

				mockPayment.EXPECT().GetPayment(paymentsAPIUrl+"/payments/"+paymentResourceID, svc.Client, apiKey).DoAndReturn(func(paymentAPIURL string, HTTPClient *http.Client, apiKey string) (data.PaymentResponse, int, error) {
					endConsumerProcess(svc, c)

					return pr, 200, nil
				})

				Convey("But payment details are never fetched", func() {
					mockPayment.EXPECT().GetPaymentDetails(paymentsAPIUrl+"/private/payments/"+paymentResourceID+"/payment-details", svc.Client, apiKey).Times(0)

					Convey("And no Eshu resource is ever constructed", func() {
						mockTransformer.EXPECT().GetEshuResource(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

						Convey("Nor is a transactions resource created", func() {
							mockTransformer.EXPECT().GetTransactionResource(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

							svc.Start(wg, c)
						})
					})
				})
			})
		})
	})

	Convey("Successful process of a single Kafka message for an 'orderable-item' payment", t, func() {
		processingOfPaymentKafkaMessageCreatesReconciliationRecords(ctrl, productMap, data.OrderableItem)
	})
}

func TestUnitMaskSensitiveFields(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockPayment := payment.NewMockFetcher(ctrl)
	mockTransformer := transformer.NewMockTransformer(ctrl)
	mockDao := dao.NewMockDAO(ctrl)

	productMap, err := createProductMap()
	if err != nil {
		log.Error(fmt.Errorf("error initialising productMap: %s", err), nil)
	}

	svc := createMockService(productMap, mockPayment, mockTransformer, mockDao)

	created := data.Created{
		Email: "test@ch.gov.uk",
	}

	cost := data.Cost{
		ClassOfPayment: []string{data.DataMaintenance},
		ProductType:    "pro-app-1",
	}

	pdr := data.PaymentResponse{
		CompanyNumber: "123456",
		CreatedBy:     created,
		Costs:         []data.Cost{cost},
	}

	Convey("test successful masking of sensitive fields ", t, func() {
		svc.MaskSensitiveFields(&pdr)

		So(pdr.CompanyNumber, ShouldEqual, "")
		So(pdr.CreatedBy.Email, ShouldEqual, "")
	})

}

func TestUnitDoNotMaskNormalFields(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockPayment := payment.NewMockFetcher(ctrl)
	mockTransformer := transformer.NewMockTransformer(ctrl)
	mockDao := dao.NewMockDAO(ctrl)

	productMap, err := createProductMap()
	if err != nil {
		log.Error(fmt.Errorf("error initialising productMap: %s", err), nil)
	}

	svc := createMockService(productMap, mockPayment, mockTransformer, mockDao)

	created := data.Created{
		Email: "test@ch.gov.uk",
	}

	cost := data.Cost{
		ClassOfPayment: []string{data.DataMaintenance},
		ProductType:    "cic-report",
	}

	pdr := data.PaymentResponse{
		CompanyNumber: "123456",
		CreatedBy:     created,
		Costs:         []data.Cost{cost},
	}

	Convey("test successful masking of sensitive fields ", t, func() {
		svc.MaskSensitiveFields(&pdr)

		So(pdr.CompanyNumber, ShouldEqual, "123456")
		So(pdr.CreatedBy.Email, ShouldEqual, "test@ch.gov.uk")
	})

}

func TestUnitCertifiedCopies(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	productMap, err := createProductMap()
	if err != nil {
		log.Error(fmt.Errorf("error initialising productMap: %s", err), nil)
	}

	Convey("Successful process of a single Kafka message for a certified copies 'orderable-item' payment with multiple costs", t, func() {
		processingOfCertifiedCopiesPaymentKafkaMessageCreatesReconciliationRecords(ctrl, productMap)
	})
}

// endConsumerProcess facilitates service termination
func endConsumerProcess(svc *Service, c chan os.Signal) {

	// Decrement the offset to escape an endless loop in the service
	svc.StopAtOffset = int64(-2)

	// Send a kill command to the input channel to terminate program execution
	go func() {
		c <- os.Kill
		close(c)
	}()
}

// processingOfPaymentKafkaMessageCreatesReconciliationRecords asserts that a payment of the class specified
// will result in a call to get the payment details and the creation of eshu and payment_transaction
// reconciliation records.
func processingOfPaymentKafkaMessageCreatesReconciliationRecords(
	ctrl *gomock.Controller,
	productMap *config.ProductMap,
	classOfPayment string) {

	wg := &sync.WaitGroup{}
	wg.Add(1)
	c := make(chan os.Signal)

	mockPayment := payment.NewMockFetcher(ctrl)
	mockTransformer := transformer.NewMockTransformer(ctrl)
	mockDao := dao.NewMockDAO(ctrl)

	svc := createMockService(productMap, mockPayment, mockTransformer, mockDao)

	Convey("Given a message is readily available for the service to consume", func() {

		svc.Consumer = createMockConsumerWithMessage()

		Convey("When the payment corresponding to the message is fetched successfully", func() {

			cost := data.Cost{
				ClassOfPayment: []string{classOfPayment},
			}

			pr := data.PaymentResponse{
				CompanyNumber: "123456",
				Costs:         []data.Cost{cost},
			}

			mockPayment.EXPECT().GetPayment(paymentsAPIUrl+"/payments/"+paymentResourceID, svc.Client, apiKey).Return(pr, 200, nil).Times(1)

			Convey("And the payment details corresponding to the message are fetched successfully", func() {

				pdr := data.PaymentDetailsResponse{
					PaymentStatus: "accepted",
				}
				mockPayment.EXPECT().GetPaymentDetails(paymentsAPIUrl+"/private/payments/"+paymentResourceID+"/payment-details", svc.Client, apiKey).Return(pdr, 200, nil).Times(1)

				Convey("Then an Eshu resource is constructed", func() {

					ers := []models.EshuResourceDao{{}}
					mockTransformer.EXPECT().GetEshuResource(pr, pdr, paymentResourceID).Return(ers, nil).Times(1)

					Convey("And committed to the DB successfully", func() {
						mockDao.EXPECT().CreateEshuResource(&ers[0]).Return(nil).Times(1)

						Convey("And a payment transactions resource is constructed", func() {

							ptrs := []models.PaymentTransactionsResourceDao{{}}
							mockTransformer.EXPECT().
								GetTransactionResource(pr, pdr, paymentResourceID).Return(ptrs, nil).Times(1)

							Convey("Which is also committed to the DB successfully", func() {

								mockDao.EXPECT().CreatePaymentTransactionsResource(&ptrs[0]).DoAndReturn(func(ptr *models.PaymentTransactionsResourceDao) error {

									// Since this is the last thing the service does, we send a signal to kill the consumer process gracefully
									endConsumerProcess(svc, c)
									return nil
								})

								svc.Start(wg, c)
							})
						})
					})
				})
			})
		})
	})
}

// processingOfCertifiedCopiesPaymentKafkaMessageCreatesReconciliationRecords asserts that a payment of the class specified
// will result in a call to get the payment details and the creation of eshu and payment_transaction
// reconciliation records capturing all of the costs involved.
func processingOfCertifiedCopiesPaymentKafkaMessageCreatesReconciliationRecords(
	ctrl *gomock.Controller,
	productMap *config.ProductMap) {

	paymentsAPI := payment.Fetch{}
	mockHttpClient := testutil.
		CreateMockClient(true, 200, testutil.CertifiedCopiesOrderGetPaymentSessionResponse)
	paymentResponse, status, _ := paymentsAPI.GetPayment("http://test-url.com", mockHttpClient, "")
	expectedNumberOfFilingHistoryDocumentCosts := len(paymentResponse.Costs)
	waitOnExpectedTransactionCreationsAndServiceShutdown := expectedNumberOfFilingHistoryDocumentCosts + 1

	wg := &sync.WaitGroup{}
	wg.Add(waitOnExpectedTransactionCreationsAndServiceShutdown)
	c := make(chan os.Signal)

	mockPayment := payment.NewMockFetcher(ctrl)
	mockDao := dao.NewMockDAO(ctrl)

	svc := createMockServiceWithRealTransformer(productMap, mockPayment, mockDao)

	Convey("Given a message is readily available for the service to consume", func() {

		svc.Consumer = createMockConsumerWithMessage()

		Convey("When the payment corresponding to the message is fetched successfully", func() {

			mockPayment.EXPECT().
				GetPayment(paymentsAPIUrl+"/payments/"+paymentResourceID, svc.Client, apiKey).
				Return(paymentResponse, status, nil).
				Times(1)

			Convey("And the payment details corresponding to the message are fetched successfully", func() {

				paymentDetailsResponse := data.PaymentDetailsResponse{
					PaymentStatus:   "accepted",
					TransactionDate: "2020-07-27T09:07:12.864Z",
				}
				mockPayment.EXPECT().
					GetPaymentDetails(paymentsAPIUrl+"/private/payments/"+paymentResourceID+"/payment-details",
						svc.Client,
						apiKey).
					Return(paymentDetailsResponse, 200, nil).
					Times(1)

				Convey("Then Eshu (product) resources are constructed", func() {

					expectedTransactionDate, _ := time.Parse(time.RFC3339Nano, paymentDetailsResponse.TransactionDate)

					Convey("And committed to the DB successfully", func() {

						expectProductsToBeCreated(
							mockDao,
							expectedTransactionDate,
							paymentResponse.Costs,
							expectedNumberOfFilingHistoryDocumentCosts)

						Convey("And payment transaction resources are constructed", func() {

							Convey("Which are also committed to the DB successfully", func() {

								expectTransactionsToBeCreated(
									wg,
									mockDao,
									expectedTransactionDate,
									paymentResponse.Costs,
									c,
									svc)

								log.Info("Starting service under test")
								go svc.Start(wg, c)
								log.Info("Waiting for all expected actions to complete")
								if waitTimeout(wg, timeout) {
									log.Error(
										fmt.Errorf("Timed out waiting %d seconds for expected actions",
											waitTimeoutSeconds))
								} else {
									log.Info("Finished waiting for all expected actions")
								}
							})
						})
					})
				})
			})
		})
	})
}

func expectProductsToBeCreated(
	mockDao *dao.MockDAO,
	expectedTransactionDate time.Time,
	expectedCosts []data.Cost,
	expectedNumberOfFilingHistoryDocumentCosts int) {
	mockDao.EXPECT().
		CreateEshuResource(expectedProduct(expectedTransactionDate, expectedProductCode(expectedCosts[0]))).
		Return(nil).
		Times(1)

	mockDao.EXPECT().
		CreateEshuResource(expectedProduct(expectedTransactionDate, expectedProductCode(expectedCosts[1]))).
		Return(nil).
		Times(expectedNumberOfFilingHistoryDocumentCosts - 1)

}

func expectedProduct(expectedTransactionDate time.Time, expectedProductCode int) *models.EshuResourceDao {
	return &models.EshuResourceDao{
		PaymentRef:      "XpaymentResourceID",
		ProductCode:     expectedProductCode,
		CompanyNumber:   "00006400",
		FilingDate:      "",
		MadeUpdate:      "",
		TransactionDate: expectedTransactionDate,
	}
}

func expectedProductCode(cost data.Cost) int {
	expectedProductCode := 27000
	if cost.ProductType == "certified-copy-incorporation-same-day" {
		expectedProductCode = 27010
	}
	return expectedProductCode
}

func expectTransactionsToBeCreated(wg *sync.WaitGroup,
	mockDao *dao.MockDAO,
	expectedTransactionDate time.Time,
	expectedCosts []data.Cost,
	c chan os.Signal,
	svc *Service) {

	mockDao.EXPECT().
		CreatePaymentTransactionsResource(
			expectedTransaction(expectedTransactionDate, expectedCosts[0].Amount)).
		DoAndReturn(func(ptr *models.PaymentTransactionsResourceDao) error {
			log.Info("CreatePaymentTransactionsResource() invocation 0")
			wg.Done()
			return nil
		}).
		Times(1)

	mockDao.EXPECT().
		CreatePaymentTransactionsResource(
			expectedTransaction(expectedTransactionDate, expectedCosts[1].Amount)).
		DoAndReturn(func(ptr *models.PaymentTransactionsResourceDao) error {
			log.Info("CreatePaymentTransactionsResource() invocation 1")
			wg.Done()
			return nil
		}).
		Times(1)

	mockDao.EXPECT().
		CreatePaymentTransactionsResource(
			expectedTransaction(expectedTransactionDate, expectedCosts[2].Amount)).
		DoAndReturn(func(ptr *models.PaymentTransactionsResourceDao) error {
			log.Info("CreatePaymentTransactionsResource() invocation 2")
			wg.Done()
			return nil
		}).
		Times(1)

	mockDao.EXPECT().
		CreatePaymentTransactionsResource(
			expectedTransaction(expectedTransactionDate, expectedCosts[3].Amount)).
		DoAndReturn(func(ptr *models.PaymentTransactionsResourceDao) error {
			log.Info("CreatePaymentTransactionsResource() final invocation (3) - closing consumer")
			wg.Done()
			// Since this is the last thing the service does, we send a signal to kill
			// the consumer process gracefully
			endConsumerProcess(svc, c)
			return nil
		}).
		Times(1)
}

func expectedTransaction(expectedTransactionDate time.Time, expectedCost string) *models.PaymentTransactionsResourceDao {
	return &models.PaymentTransactionsResourceDao{
		TransactionID:     "XpaymentResourceID",
		TransactionDate:   expectedTransactionDate,
		Email:             "demo@ch.gov.uk",
		PaymentMethod:     "GovPay",
		Amount:            expectedCost,
		CompanyNumber:     "00006400",
		TransactionType:   "Immediate bill",
		OrderReference:    "Payments reconciliation testing payment session ref GCI-1312",
		Status:            "accepted",
		UserID:            "system",
		OriginalReference: "",
		DisputeDetails:    "",
	}
}

// waitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
// See https://stackoverflow.com/questions/32840687/timeout-for-waitgroup-wait.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
