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
)

const paymentsAPIUrl = "paymentsAPIUrl"
const apiKey = "apiKey"
const paymentResourceID = "paymentResourceID"

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
	}
}

func createMockConsumerWithMessage() *consumer.GroupConsumer {

	return &consumer.GroupConsumer{
		GConsumer: MockConsumer{},
	}
}

func createMockProducer() *producer.Producer {

	return &producer.Producer{
		SyncProducer: MockProducer{},
	}
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

func TestUnitStart(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	productMap, err := createProductMap()
	if err != nil {
		log.Error(fmt.Errorf("error initialising productMap: %s", err), nil)
	}

	Convey("Successful process of a single Kafka message for a 'data-maintenance' payment", t, func() {

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
					ClassOfPayment: []string{"data-maintenance"},
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

						var er models.EshuResourceDao
						mockTransformer.EXPECT().GetEshuResource(pr, pdr, paymentResourceID).Return(er, nil).Times(1)

						Convey("And committed to the DB successfully", func() {
							mockDao.EXPECT().CreateEshuResource(&er).Return(nil).Times(1)

							Convey("And a payment transactions resource is constructed", func() {

								var ptr models.PaymentTransactionsResourceDao
								mockTransformer.EXPECT().GetTransactionResource(pr, pdr, paymentResourceID).Return(ptr, nil).Times(1)

								Convey("Which is also committed to the DB successfully", func() {

									mockDao.EXPECT().CreatePaymentTransactionsResource(&ptr).DoAndReturn(func(ptr *models.PaymentTransactionsResourceDao) error {

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
					ClassOfPayment: []string{"penalty"},
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
					ClassOfPayment: []string{"orderable-item"},
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

						var er models.EshuResourceDao
						mockTransformer.EXPECT().GetEshuResource(pr, pdr, paymentResourceID).Return(er, nil).Times(1)

						Convey("And committed to the DB successfully", func() {
							mockDao.EXPECT().CreateEshuResource(&er).Return(nil).Times(1)

							Convey("And a payment transactions resource is constructed", func() {

								var ptr models.PaymentTransactionsResourceDao
								mockTransformer.EXPECT().GetTransactionResource(pr, pdr, paymentResourceID).Return(ptr, nil).Times(1)

								Convey("Which is also committed to the DB successfully", func() {

									mockDao.EXPECT().CreatePaymentTransactionsResource(&ptr).DoAndReturn(func(ptr *models.PaymentTransactionsResourceDao) error {

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
		ClassOfPayment: []string{"data-maintenance"},
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
		ClassOfPayment: []string{"data-maintenance"},
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
