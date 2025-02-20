package service

import (
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/companieshouse/chs.go/avro"
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
const differentPaymentResourceID = "differentPaymentResourceID"
const refundID = "refundId"
const productsLogKey = "products"

var (
	handleErrorCalled bool = false
)

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
		HandleError: func(err error, offset int64, str interface{}) error {
			handleErrorCalled = true
			return err
		},
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
func createMockConsumerWithPaymentMessage(paymentId string) *consumer.GroupConsumer {
	return createMockConsumerWithMessage(paymentId, "")
}

func createMockConsumerWithRefundMessage(paymentId string, refundId string) *consumer.GroupConsumer {
	return createMockConsumerWithMessage(paymentId, refundId)
}

func createMockConsumerWithMessage(paymentId string, refundId string) *consumer.GroupConsumer {

	return &consumer.GroupConsumer{
		GConsumer: MockConsumer{PaymentId: paymentId, RefundId: refundId},
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

	log.Info("Product map config has been loaded", log.Data{productsLogKey: productMap})

	return productMap, nil
}

func getDefaultSchema() string {

	return "{\"type\":\"record\",\"name\":\"payment_processed\",\"namespace\":\"payments\",\"fields\":[{\"name\":\"payment_resource_id\",\"type\":\"string\"}, {\"name\":\"refund_id\",\"type\":\"string\"}]}"
}

var MockSchema = &avro.Schema{
	Definition: getDefaultSchema(),
}

type MockProducer struct {
	sarama.SyncProducer
}

func (m MockProducer) Close() error {
	return nil
}

type MockConsumer struct {
	PaymentId string
	RefundId  string
}

func (m MockConsumer) prepareTestKafkaMessage() ([]byte, error) {
	return MockSchema.Marshal(data.PaymentProcessed{ResourceURI: m.PaymentId, RefundId: m.RefundId})
}

func (m MockConsumer) Close() error {
	return nil
}

func (m MockConsumer) Messages() <-chan *sarama.ConsumerMessage {
	out := make(chan *sarama.ConsumerMessage)

	bytes, _ := m.prepareTestKafkaMessage()
	go func() {
		out <- &sarama.ConsumerMessage{
			Value: bytes,
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
			svc.Consumer = createMockConsumerWithPaymentMessage(paymentResourceID)

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
						mockTransformer.EXPECT().GetEshuResources(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

						Convey("Nor is a transactions resource created", func() {
							mockTransformer.EXPECT().GetTransactionResources(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

							svc.Start(wg, c)
						})
					})
				})
			})
		})
	})

	Convey("Process of a single Kafka message for a 'penalty-lfp' payment", t, func() {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		c := make(chan os.Signal)

		mockPayment := payment.NewMockFetcher(ctrl)
		mockTransformer := transformer.NewMockTransformer(ctrl)
		mockDao := dao.NewMockDAO(ctrl)

		svc := createMockService(productMap, mockPayment, mockTransformer, mockDao)

		Convey("Given a message is readily available for the service to consume", func() {
			svc.Consumer = createMockConsumerWithPaymentMessage(paymentResourceID)

			Convey("When the payment corresponding to the message is fetched successfully", func() {

				cost := data.Cost{
					ClassOfPayment: []string{data.PenaltyLfp},
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
						mockTransformer.EXPECT().GetEshuResources(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

						Convey("Nor is a transactions resource created", func() {
							mockTransformer.EXPECT().GetTransactionResources(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

							svc.Start(wg, c)
						})
					})
				})
			})
		})
	})

	Convey("Successful process of a single Kafka message for an 'penalty-sanctions' payment", t, func() {
		processingOfPaymentKafkaMessageCreatesReconciliationRecords(ctrl, productMap, data.PenaltySanctions)
	})

	Convey("Successful process of a single Kafka message for an 'orderable-item' payment", t, func() {
		processingOfPaymentKafkaMessageCreatesReconciliationRecords(ctrl, productMap, data.OrderableItem)
	})

	Convey("Successful process of a single Kafka message for a refund", t, func() {
		processingOfRefundKafkaMessageCreatesReconciliationRecords(ctrl, productMap, data.OrderableItem)
	})

	Convey("Successful process of a single Kafka message for a refund which was submitted", t, func() {
		processingOfRefundKafkaMessageWithSubmittedStatusCreatesReconciliationRecords(ctrl, productMap, data.OrderableItem)
	})

	Convey("Successful process of a single Kafka message for payment with multiple refunds", t, func() {
		processingOfRefundKafkaMessageWithMultipleRefundsCreatesReconciliationRecords(ctrl, productMap, data.OrderableItem)
	})

	Convey("Unsuccessful process of a single Kafka message for a refund", t, func() {
		processingOfUnsuccessfulRefundKafkaMessageDoesNotCreateReconciliationRecords(ctrl, productMap, data.OrderableItem)
	})

	Convey("Unsuccessful process of a single Kafka message for a refund which was submitted", t, func() {
		processingOfUnsuccessfulRefundKafkaMessageWithSubmittedDoesNotCreateReconciliationRecords(ctrl, productMap, data.OrderableItem)
	})

	Convey("Unsuccessful process of a single Kafka message for a refund with payment not having correct refund id", t, func() {
		processingOfUnsuccessfulRefundKafkaMessageWithIncorrectRefundIdDoesNotCreateReconciliationRecords(ctrl, productMap, data.OrderableItem)
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

	Convey("Successful process of a single message for a certified copies 'orderable-item' payment with multiple costs",
		t, func() {
			processingOfCertifiedCopiesPaymentKafkaMessageCreatesReconciliationRecords(
				ctrl,
				productMap,
				testutil.CertifiedCopiesOrderGetPaymentSessionResponse)
		})

	Convey("Successful process of a single message for a certified copies 'orderable-item' payment with a single cost",
		t, func() {
			processingOfCertifiedCopiesPaymentKafkaMessageCreatesReconciliationRecords(
				ctrl,
				productMap,
				testutil.CertifiedCopiesSingleCostOrderGetPaymentSessionResponse)
		})

	Convey("HandleError invoked to handle errors", t, func() {

		mockPayment := payment.NewMockFetcher(ctrl)
		mockTransformer := transformer.NewMockTransformer(ctrl)
		mockDao := dao.NewMockDAO(ctrl)

		svc := createMockService(productMap, mockPayment, mockTransformer, mockDao)

		message := sarama.ConsumerMessage{Offset: 1}
		paymentResponse := data.PaymentResponse{}
		details := data.PaymentDetailsResponse{}
		pp := data.PaymentProcessed{
			ResourceURI: "paymentResponse ID",
		}
		mockError := errors.New("test-simulated mock error")

		Convey("HandleError invoked to handle error creating eshus", func() {

			// Given
			handleErrorCalled = false
			mockTransformer.EXPECT().
				GetEshuResources(paymentResponse, details, pp.ResourceURI).
				Return(nil, mockError).
				Times(1)

			// When
			svc.getEshuResources(&message, paymentResponse, details, pp)

			// Then
			So(handleErrorCalled, ShouldEqual, true)

		})

		Convey("HandleError invoked to handle error saving eshus", func() {

			// Given
			handleErrorCalled = false
			eshus := []models.EshuResourceDao{{}}
			mockDao.EXPECT().CreateEshuResource(&eshus[0]).Return(mockError).Times(1)

			// When
			svc.saveEshuResources(&message, eshus, pp)

			// Then
			So(handleErrorCalled, ShouldEqual, true)
		})

		Convey("HandleError invoked to handle error creating transactions", func() {

			// Given
			handleErrorCalled = false
			mockTransformer.EXPECT().
				GetTransactionResources(paymentResponse, details, pp.ResourceURI).
				Return(nil, mockError).
				Times(1)

			// When
			svc.getTransactionResources(&message, paymentResponse, details, pp)

			// Then
			So(handleErrorCalled, ShouldEqual, true)

		})

		Convey("HandleError invoked to handle error saving transactions", func() {

			// Given
			handleErrorCalled = false
			txns := []models.PaymentTransactionsResourceDao{{}}
			mockDao.EXPECT().CreatePaymentTransactionsResource(&txns[0]).Return(mockError).Times(1)

			// When
			svc.saveTransactionResources(&message, txns, pp)

			// Then
			So(handleErrorCalled, ShouldEqual, true)
		})
	})

}

func TestUnitCheckSkipGoneResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	productMap, err := createProductMap()
	if err != nil {
		log.Error(fmt.Errorf("error initialising productMap: %s", err), nil)
	}

	message := &sarama.ConsumerMessage{Offset: 1, Topic: "test topic"}

	mockPayment := payment.NewMockFetcher(ctrl)
	mockTransformer := transformer.NewMockTransformer(ctrl)
	mockDao := dao.NewMockDAO(ctrl)

	svc := createMockService(productMap, mockPayment, mockTransformer, mockDao)

	Convey("Given SkipGoneResource is true", t, func() {
		svc.SkipGoneResource = true
		Convey("When SkipGoneResourceId is empty", func() {
			svc.SkipGoneResourceId = ""
			Convey("Then checkSkipGoneResource should return true", func() {
				So(svc.checkSkipGoneResource(paymentResourceID, message), ShouldEqual, true)
			})
		})
		Convey("When SkipGoneResourceId is set", func() {
			svc.SkipGoneResourceId = paymentResourceID
			Convey("Then checkSkipGoneResource should return true when the payment ids match", func() {
				So(svc.checkSkipGoneResource(paymentResourceID, message), ShouldEqual, true)
			})
			Convey("Then checkSkipGoneResource should return false when the payment ids do not match", func() {
				So(svc.checkSkipGoneResource(differentPaymentResourceID, message), ShouldEqual, false)
			})
		})
	})

	Convey("Given SkipGoneResource is false", t, func() {
		svc.SkipGoneResource = false
		Convey("When SkipGoneResourceId is empty", func() {
			svc.SkipGoneResourceId = ""
			Convey("Then checkSkipGoneResource should return false", func() {
				So(svc.checkSkipGoneResource(paymentResourceID, message), ShouldEqual, false)
			})
		})
		Convey("When SkipGoneResourceId is set", func() {
			svc.SkipGoneResourceId = paymentResourceID
			Convey("Then checkSkipGoneResource should return false when the payment ids match", func() {
				So(svc.checkSkipGoneResource(paymentResourceID, message), ShouldEqual, false)
			})
			Convey("Then checkSkipGoneResource should return false when the payment ids do not match", func() {
				So(svc.checkSkipGoneResource(differentPaymentResourceID, message), ShouldEqual, false)
			})
		})
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

		svc.Consumer = createMockConsumerWithPaymentMessage(paymentResourceID)

		Convey("When the payment corresponding to the message is fetched successfully", func() {

			cost := data.Cost{
				ClassOfPayment: []string{classOfPayment},
				ProductType:    "certificate",
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
					mockTransformer.EXPECT().GetEshuResources(pr, pdr, paymentResourceID).Return(ers, nil).Times(1)

					Convey("And committed to the DB successfully", func() {
						mockDao.EXPECT().CreateEshuResource(&ers[0]).Return(nil).Times(1)

						Convey("And a payment transactions resource is constructed", func() {

							ptrs := []models.PaymentTransactionsResourceDao{{}}
							mockTransformer.EXPECT().
								GetTransactionResources(pr, pdr, paymentResourceID).Return(ptrs, nil).Times(1)

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

// processingOfRefundKafkaMessageCreatesReconciliationRecords asserts that a refund of the class specified
// will result in a call to get the payment details and the creation of refund reconciliation records.
func processingOfRefundKafkaMessageCreatesReconciliationRecords(
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

		svc.Consumer = createMockConsumerWithRefundMessage(paymentResourceID, refundID)

		Convey("When the payment corresponding to the message is fetched successfully", func() {

			cost := data.Cost{
				ClassOfPayment: []string{classOfPayment},
				ProductType:    "certificate",
			}

			pr := data.PaymentResponse{
				CompanyNumber: "123456",
				Costs:         []data.Cost{cost},
				Refunds: []data.RefundResource{{
					RefundId:          refundID,
					CreatedAt:         "",
					Amount:            0,
					Status:            "success",
					ExternalRefundUrl: "",
				}},
			}

			mockPayment.EXPECT().GetPayment(paymentsAPIUrl+"/payments/"+paymentResourceID, svc.Client, apiKey).Return(pr, 200, nil).Times(1)

			Convey("And the payment details corresponding to the message are fetched successfully", func() {

				pdr := data.PaymentDetailsResponse{
					PaymentStatus: "accepted",
				}
				mockPayment.EXPECT().GetPaymentDetails(paymentsAPIUrl+"/private/payments/"+paymentResourceID+"/payment-details", svc.Client, apiKey).Return(pdr, 200, nil).Times(1)

				Convey("Then a Refund resource is constructed", func() {

					refund := models.RefundResourceDao{}
					mockTransformer.EXPECT().GetRefundResource(pr, pr.Refunds[0], paymentResourceID).Return(refund, nil).Times(1)

					Convey("And committed to the DB successfully", func() {
						mockDao.EXPECT().CreateRefundResource(&refund).DoAndReturn(func(ptr *models.RefundResourceDao) error {

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
}

// processingOfRefundKafkaMessageWithSubmittedStatusCreatesReconciliationRecords asserts that a refund of the class specified
// with submitted status will result in a call to update refund status and get the payment details and the creation of refund reconciliation records.
func processingOfRefundKafkaMessageWithSubmittedStatusCreatesReconciliationRecords(
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

		svc.Consumer = createMockConsumerWithRefundMessage(paymentResourceID, refundID)

		Convey("When the payment corresponding to the message is fetched successfully", func() {

			cost := data.Cost{
				ClassOfPayment: []string{classOfPayment},
				ProductType:    "certificate",
			}

			pr := data.PaymentResponse{
				CompanyNumber: "123456",
				Costs:         []data.Cost{cost},
				Refunds: []data.RefundResource{{
					RefundId:          refundID,
					CreatedAt:         "",
					Amount:            0,
					Status:            "submitted",
					ExternalRefundUrl: "",
				}},
			}

			mockPayment.EXPECT().GetPayment(paymentsAPIUrl+"/payments/"+paymentResourceID, svc.Client, apiKey).Return(pr, 200, nil).Times(1)

			Convey("And the payment details corresponding to the message are fetched successfully", func() {

				pdr := data.PaymentDetailsResponse{
					PaymentStatus: "accepted",
				}
				mockPayment.EXPECT().GetPaymentDetails(paymentsAPIUrl+"/private/payments/"+paymentResourceID+"/payment-details", svc.Client, apiKey).Return(pdr, 200, nil).Times(1)

				Convey("Then a Refund status is fetched", func() {
					refundResource := data.RefundResource{
						RefundId:          refundID,
						CreatedAt:         "",
						Amount:            0,
						Status:            "success",
						ExternalRefundUrl: "",
					}

					mockPayment.EXPECT().GetLatestRefundStatus(paymentsAPIUrl+"/payments/"+paymentResourceID+"/refunds/"+refundID, svc.Client, apiKey).Return(&refundResource, 200, nil).Times(1)

					Convey("Then a Refund resource is constructed", func() {

						refund := models.RefundResourceDao{}
						mockTransformer.EXPECT().GetRefundResource(pr, refundResource, paymentResourceID).Return(refund, nil).Times(1)

						Convey("And committed to the DB successfully", func() {
							mockDao.EXPECT().CreateRefundResource(&refund).DoAndReturn(func(ptr *models.RefundResourceDao) error {

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
}

// processingOfRefundKafkaMessageWithMultipleRefundsCreatesReconciliationRecords asserts that a refund of the class specified
// will result in a call to get the payment details and the creation of refund reconciliation records with proper refund.
func processingOfRefundKafkaMessageWithMultipleRefundsCreatesReconciliationRecords(
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

		svc.Consumer = createMockConsumerWithRefundMessage(paymentResourceID, refundID)

		Convey("When the payment corresponding to the message is fetched successfully", func() {

			cost := data.Cost{
				ClassOfPayment: []string{classOfPayment},
				ProductType:    "certificate",
			}

			pr := data.PaymentResponse{
				CompanyNumber: "123456",
				Costs:         []data.Cost{cost},
				Refunds: []data.RefundResource{
					{
						RefundId:          refundID,
						CreatedAt:         "",
						Amount:            0,
						Status:            "success",
						ExternalRefundUrl: "",
					},
					{
						RefundId:          refundID + "1",
						CreatedAt:         "",
						Amount:            0,
						Status:            "failed",
						ExternalRefundUrl: "",
					},
				},
			}

			mockPayment.EXPECT().GetPayment(paymentsAPIUrl+"/payments/"+paymentResourceID, svc.Client, apiKey).Return(pr, 200, nil).Times(1)

			Convey("And the payment details corresponding to the message are fetched successfully", func() {

				pdr := data.PaymentDetailsResponse{
					PaymentStatus: "accepted",
				}
				mockPayment.EXPECT().GetPaymentDetails(paymentsAPIUrl+"/private/payments/"+paymentResourceID+"/payment-details", svc.Client, apiKey).Return(pdr, 200, nil).Times(1)

				Convey("Then a Refund resource is constructed", func() {

					refund := models.RefundResourceDao{}
					mockTransformer.EXPECT().GetRefundResource(pr, pr.Refunds[0], paymentResourceID).Return(refund, nil).Times(1)

					Convey("And committed to the DB successfully", func() {
						mockDao.EXPECT().CreateRefundResource(&refund).DoAndReturn(func(ptr *models.RefundResourceDao) error {

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
}

// processingOfUnsuccessfulRefundKafkaMessageWithIncorrectRefundIdDoesNotCreateReconciliationRecords asserts that
// a refund of the class specified will not result in a call to get the payment details and will not create the refund
// reconciliation records.
func processingOfUnsuccessfulRefundKafkaMessageWithIncorrectRefundIdDoesNotCreateReconciliationRecords(
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

		svc.Consumer = createMockConsumerWithRefundMessage(paymentResourceID, refundID)

		Convey("When the payment corresponding to the message is fetched successfully", func() {

			cost := data.Cost{
				ClassOfPayment: []string{classOfPayment},
				ProductType:    "certificate",
			}

			pr := data.PaymentResponse{
				CompanyNumber: "123456",
				Costs:         []data.Cost{cost},
				Refunds: []data.RefundResource{{
					RefundId:          refundID + "x",
					CreatedAt:         "",
					Amount:            0,
					Status:            "failed",
					ExternalRefundUrl: "",
				}},
			}

			mockPayment.EXPECT().GetPayment(paymentsAPIUrl+"/payments/"+paymentResourceID, svc.Client, apiKey).Return(pr, 200, nil).Times(1)

			Convey("And the payment details corresponding to the message are fetched successfully", func() {

				pdr := data.PaymentDetailsResponse{
					PaymentStatus: "accepted",
				}
				mockPayment.EXPECT().GetPaymentDetails(paymentsAPIUrl+"/private/payments/"+paymentResourceID+"/payment-details", svc.Client, apiKey).DoAndReturn(func(paymentAPIURL string, HTTPClient *http.Client, apiKey string) (data.PaymentDetailsResponse, int, error) {
					endConsumerProcess(svc, c)

					return pdr, 200, nil

				}).Times(1)

				Convey("Then a Refund resource is not constructed", func() {

					mockTransformer.EXPECT().GetRefundResource(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

					Convey("And not committed to the DB", func() {
						mockDao.EXPECT().CreateRefundResource(gomock.Any()).Times(0)

						svc.Start(wg, c)
					})
				})
			})
		})
	})
}

// processingOfUnsuccessfulRefundKafkaMessageDoesNotCreateReconciliationRecords asserts that a refund of the class specified
// will not result in a call to get the payment details and will not create the refund reconciliation records.
func processingOfUnsuccessfulRefundKafkaMessageDoesNotCreateReconciliationRecords(
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

		svc.Consumer = createMockConsumerWithRefundMessage(paymentResourceID, refundID)

		Convey("When the payment corresponding to the message is fetched successfully", func() {

			cost := data.Cost{
				ClassOfPayment: []string{classOfPayment},
				ProductType:    "certificate",
			}

			pr := data.PaymentResponse{
				CompanyNumber: "123456",
				Costs:         []data.Cost{cost},
				Refunds: []data.RefundResource{{
					RefundId:          refundID,
					CreatedAt:         "",
					Amount:            0,
					Status:            "failed",
					ExternalRefundUrl: "",
				}},
			}

			mockPayment.EXPECT().GetPayment(paymentsAPIUrl+"/payments/"+paymentResourceID, svc.Client, apiKey).Return(pr, 200, nil).Times(1)

			Convey("And the payment details corresponding to the message are fetched successfully", func() {

				pdr := data.PaymentDetailsResponse{
					PaymentStatus: "accepted",
				}
				mockPayment.EXPECT().GetPaymentDetails(paymentsAPIUrl+"/private/payments/"+paymentResourceID+"/payment-details", svc.Client, apiKey).DoAndReturn(func(paymentAPIURL string, HTTPClient *http.Client, apiKey string) (data.PaymentDetailsResponse, int, error) {
					endConsumerProcess(svc, c)

					return pdr, 200, nil

				}).Times(1)

				Convey("Then a Refund resource is not constructed", func() {

					mockTransformer.EXPECT().GetRefundResource(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

					Convey("And not committed to the DB", func() {
						mockDao.EXPECT().CreateRefundResource(gomock.Any()).Times(0)

						svc.Start(wg, c)
					})
				})
			})
		})
	})
}

// processingOfUnsuccessfulRefundKafkaMessageDoesNotCreateReconciliationRecords asserts that a refund of the class specified
// will not result in a call to get the payment details and will not create the refund reconciliation records.
func processingOfUnsuccessfulRefundKafkaMessageWithSubmittedDoesNotCreateReconciliationRecords(
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

		svc.Consumer = createMockConsumerWithRefundMessage(paymentResourceID, refundID)

		Convey("When the payment corresponding to the message is fetched successfully", func() {

			cost := data.Cost{
				ClassOfPayment: []string{classOfPayment},
				ProductType:    "certificate",
			}

			pr := data.PaymentResponse{
				CompanyNumber: "123456",
				Costs:         []data.Cost{cost},
				Refunds: []data.RefundResource{{
					RefundId:          refundID,
					CreatedAt:         "",
					Amount:            0,
					Status:            "submitted",
					ExternalRefundUrl: "",
				}},
			}

			mockPayment.EXPECT().GetPayment(paymentsAPIUrl+"/payments/"+paymentResourceID, svc.Client, apiKey).Return(pr, 200, nil).Times(1)

			Convey("And the payment details corresponding to the message are fetched successfully", func() {

				pdr := data.PaymentDetailsResponse{
					PaymentStatus: "accepted",
				}
				mockPayment.EXPECT().GetPaymentDetails(paymentsAPIUrl+"/private/payments/"+paymentResourceID+"/payment-details", svc.Client, apiKey).Return(pdr, 200, nil).Times(1)

				Convey("Then a Refund status is fetched", func() {
					refundResource := data.RefundResource{
						RefundId:          refundID,
						CreatedAt:         "",
						Amount:            0,
						Status:            "failed",
						ExternalRefundUrl: "",
					}

					mockPayment.EXPECT().GetLatestRefundStatus(paymentsAPIUrl+"/payments/"+paymentResourceID+"/refunds/"+refundID, svc.Client, apiKey).DoAndReturn(func(paymentAPIURL string, HTTPClient *http.Client, apiKey string) (*data.RefundResource, int, error) {
						endConsumerProcess(svc, c)

						return &refundResource, 200, nil
					}).Times(1)

					Convey("Then a Refund resource is not constructed", func() {

						mockTransformer.EXPECT().GetRefundResource(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

						Convey("And not committed to the DB", func() {
							mockDao.EXPECT().CreateRefundResource(gomock.Any()).Times(0)

							svc.Start(wg, c)
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
	productMap *config.ProductMap,
	responseBody string) {

	paymentsAPI := payment.Fetch{}
	mockHttpClient := testutil.
		CreateMockClient(true, 200, responseBody)
	paymentResponse, status, _ := paymentsAPI.GetPayment("http://test-url.com", mockHttpClient, "")
	expectedNumberOfFilingHistoryDocumentCosts := len(paymentResponse.Costs)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	c := make(chan os.Signal)

	mockPayment := payment.NewMockFetcher(ctrl)
	mockDao := dao.NewMockDAO(ctrl)

	svc := createMockServiceWithRealTransformer(productMap, mockPayment, mockDao)

	Convey("Given a message is readily available for the service to consume", func() {

		svc.Consumer = createMockConsumerWithPaymentMessage(paymentResourceID)

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
									mockDao,
									expectedTransactionDate,
									paymentResponse.Costs,
									c,
									svc)

								log.Info("Starting service under test")
								svc.Start(wg, c)
								log.Info("Completing test")
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

	if len(expectedCosts) > 1 {
		mockDao.EXPECT().
			CreateEshuResource(expectedProduct(expectedTransactionDate, expectedProductCode(expectedCosts[1]))).
			Return(nil).
			Times(expectedNumberOfFilingHistoryDocumentCosts - 1)
	}

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

func expectTransactionsToBeCreated(
	mockDao *dao.MockDAO,
	expectedTransactionDate time.Time,
	expectedCosts []data.Cost,
	c chan os.Signal,
	svc *Service) {

	numberOfCosts := len(expectedCosts)

	mockDao.EXPECT().
		CreatePaymentTransactionsResource(
			expectedTransaction(expectedTransactionDate, expectedCosts[0].Amount)).
		DoAndReturn(func(ptr *models.PaymentTransactionsResourceDao) error {
			return endConsumerProcessIfLastTransactionCreated(numberOfCosts, 0, svc, c)
		}).
		Times(1)

	if numberOfCosts > 1 {
		mockDao.EXPECT().
			CreatePaymentTransactionsResource(
				expectedTransaction(expectedTransactionDate, expectedCosts[1].Amount)).
			DoAndReturn(func(ptr *models.PaymentTransactionsResourceDao) error {
				return endConsumerProcessIfLastTransactionCreated(numberOfCosts, 1, svc, c)
			}).
			Times(1)
	} else {
		return
	}

	if numberOfCosts > 2 {
		mockDao.EXPECT().
			CreatePaymentTransactionsResource(
				expectedTransaction(expectedTransactionDate, expectedCosts[2].Amount)).
			DoAndReturn(func(ptr *models.PaymentTransactionsResourceDao) error {
				return endConsumerProcessIfLastTransactionCreated(numberOfCosts, 2, svc, c)
			}).
			Times(1)
	} else {
		return
	}

	if numberOfCosts > 3 {
		mockDao.EXPECT().
			CreatePaymentTransactionsResource(
				expectedTransaction(expectedTransactionDate, expectedCosts[3].Amount)).
			DoAndReturn(func(ptr *models.PaymentTransactionsResourceDao) error {
				return endConsumerProcessIfLastTransactionCreated(numberOfCosts, 3, svc, c)
			}).
			Times(1)
	} else {
		return
	}
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

// Ends the consumer process if the transaction for the last cost has been created.
// Returns nil error for test code legibility.
func endConsumerProcessIfLastTransactionCreated(numberOfCosts int, costIndex int, svc *Service, c chan os.Signal) error {
	log.Info(fmt.Sprintf("CreatePaymentTransactionsResource() invocation %d", costIndex+1))
	if costIndex == numberOfCosts-1 {
		// Since this is the last thing the service does, we send a signal to kill
		// the consumer process gracefully.
		log.Info(fmt.Sprintf("Closing consumer after invocation %d", costIndex+1))
		endConsumerProcess(svc, c)
	}
	return nil
}
