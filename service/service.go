package service

import (
	"errors"
	"fmt"
	"github.com/companieshouse/payment-reconciliation-consumer/dao"
	"github.com/companieshouse/payment-reconciliation-consumer/keys"
	"github.com/companieshouse/payment-reconciliation-consumer/models"
	"github.com/companieshouse/payment-reconciliation-consumer/transformer"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/companieshouse/chs.go/avro"
	"github.com/companieshouse/chs.go/avro/schema"
	"github.com/companieshouse/chs.go/kafka/client"
	"github.com/companieshouse/chs.go/kafka/consumer/cluster"
	"github.com/companieshouse/chs.go/kafka/producer"
	"github.com/companieshouse/chs.go/kafka/resilience"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payment-reconciliation-consumer/config"
	"github.com/companieshouse/payment-reconciliation-consumer/data"
	"github.com/companieshouse/payment-reconciliation-consumer/payment"
)

// Service represents service config for payment-reconciliation-consumer
type Service struct {
	Consumer        *consumer.GroupConsumer
	Producer        *producer.Producer
	PpSchema        string
	Client          *http.Client
	InitialOffset   int64
	HandleError     func(err error, offset int64, str interface{}) error
	Topic           string
	Retry           *resilience.ServiceRetry
	IsErrorConsumer bool
	BrokerAddr      []string
	APIKey          string
	PaymentsAPIURL  string
	DAO             dao.DAO
	TranCollection  string
	ProdCollection  string
	ProductMap      *config.ProductMap
	Payments        payment.Fetcher
	Transformer     transformer.Transformer
	StopAtOffset    int64
}

// New creates a new instance of service with a given consumerGroup name,
// consumerTopic, throttleRate and payment-reconciliation-consumer config
func New(consumerTopic, consumerGroupName string, cfg *config.Config, retry *resilience.ServiceRetry) (*Service, error) {

	schemaName := "payment-processed"
	ppSchema, err := schema.Get(cfg.SchemaRegistryURL, schemaName)
	if err != nil {
		log.Error(fmt.Errorf("error receiving %s schema: %s", schemaName, err))
		return nil, err
	}

	log.Info("Successfully received schema", log.Data{keys.SchemaName: schemaName})

	appName := cfg.Namespace()

	productMap, err := config.GetProductMap()
	if err != nil {
		log.Error(fmt.Errorf("error initialising productMap: %s", err), nil)
		return nil, err
	}

	p, err := producer.New(&producer.Config{Acks: &producer.WaitForAll, BrokerAddrs: cfg.BrokerAddr})
	if err != nil {
		log.Error(fmt.Errorf("error initialising producer: %s", err), nil)
		return nil, err
	}

	maxRetries := 0
	if retry != nil {
		maxRetries = retry.MaxRetries
	}

	log.Info("Start Request Create resilient Kafka service", log.Data{
		keys.BaseTopic:  consumerTopic,
		keys.AppName:    appName,
		keys.MaxRetries: maxRetries,
		keys.Producer:   p})
	rh := resilience.NewHandler(consumerTopic, "consumer", retry, p, &avro.Schema{Definition: ppSchema})

	// Work out what topic we're consuming from, depending on whether were processing resilience or error input
	topicName := consumerTopic
	if retry != nil {
		topicName = rh.GetRetryTopicName()
	}
	if cfg.IsErrorConsumer {
		topicName = rh.GetErrorTopicName()
	}

	var resetOffset bool

	consumerConfig := &consumer.Config{
		Topics:       []string{topicName},
		ZookeeperURL: cfg.ZookeeperURL,
		BrokerAddr:   cfg.BrokerAddr,
	}

	log.Info("attempting to join consumer group", log.Data{
		"consumer_group_name": consumerGroupName,
		"topic":               topicName,
	})

	groupConfig := &consumer.GroupConfig{
		GroupName:   consumerGroupName,
		ResetOffset: resetOffset,
		Chroot:      cfg.ZookeeperChroot,
	}

	c := consumer.NewConsumerGroup(consumerConfig)
	if err = c.JoinGroup(groupConfig); err != nil {
		log.Error(fmt.Errorf("error joining '"+consumerGroupName+"' consumer group", err), nil)
		return nil, err
	}

	// If we're an error consumer, then capture the tail of the topic, and only consume up to that offset.
	stopAtOffset := int64(-1)
	if cfg.IsErrorConsumer {
		stopAtOffset, err = client.TopicOffset(cfg.BrokerAddr, topicName)
		if err != nil {
			log.Error(err, log.Data{keys.Topic: topicName})
		}
		log.Info("error queue consumer will stop when backlog offset reached",
			log.Data{keys.BacklogOffset: stopAtOffset})
	}

	return &Service{
		Consumer:        c,
		Producer:        p,
		PpSchema:        ppSchema,
		Client:          &http.Client{},
		HandleError:     rh.HandleError,
		Topic:           topicName,
		Retry:           retry,
		IsErrorConsumer: cfg.IsErrorConsumer,
		BrokerAddr:      cfg.BrokerAddr,
		APIKey:          cfg.ChsAPIKey,
		PaymentsAPIURL:  cfg.PaymentsAPIURL,
		DAO:             dao.New(cfg),
		TranCollection:  cfg.TransactionsCollection,
		ProdCollection:  cfg.ProductsCollection,
		ProductMap:      productMap,
		Payments:        payment.New(),
		Transformer:     transformer.New(),
		StopAtOffset:    stopAtOffset,
	}, nil

}

// Start begins the service - Messages are consumed from the payment-processed
// topic
func (svc *Service) Start(wg *sync.WaitGroup, c chan os.Signal) {
	log.Info("service starting, consuming from the " + svc.Topic + " topic")

	var err error
	var message *sarama.ConsumerMessage

	// We want to stop the processing of the service if consuming from an
	// error queue if all messages that were initially in the queue have
	// been cleared using the stopAtOffset
	running := true
	for running && (svc.StopAtOffset == -1 || message == nil || message.Offset < svc.StopAtOffset) {

		if message != nil {
			// Commit the message we've just been processing before starting the next
			log.Trace("Committing message", log.Data{keys.Offset: message.Offset})
			svc.Consumer.MarkOffset(message, "")
			if err := svc.Consumer.CommitOffsets(); err != nil {
				log.Error(err, log.Data{keys.Offset: message.Offset})
			}
		}

		if svc.Retry != nil && svc.Retry.ThrottleRate > 0 {
			time.Sleep(svc.Retry.ThrottleRate * time.Second)
		}

		select {
		case <-c:
			running = false

		case message = <-svc.Consumer.Messages():
			// Falls into this block when a message becomes available from consumer

			if message != nil {
				if message.Offset >= svc.InitialOffset {
					log.Info("Received message from Payment Service. Attempting reconciliation...")

					// GetPayment the payment session first
					var pp data.PaymentProcessed
					paymentProcessedSchema := &avro.Schema{
						Definition: svc.PpSchema,
					}

					err = paymentProcessedSchema.Unmarshal(message.Value, &pp)
					if err != nil {
						log.Error(err, log.Data{keys.Offset: message.Offset})
						_ = svc.HandleError(err, message.Offset, &message.Value)
						continue
					}

					//Create GetPayment payment session URL
					getPaymentURL := svc.PaymentsAPIURL + "/payments/" + pp.ResourceURI
					log.Info("Payment URL : " + getPaymentURL)

					//Call GetPayment payment session from payments API
					paymentResponse, statusCode, err := svc.Payments.GetPayment(getPaymentURL, svc.Client, svc.APIKey)
					if err != nil {
						log.Error(err, log.Data{keys.Offset: message.Offset})
						_ = svc.HandleError(err, message.Offset, &paymentResponse)
					}
					log.Info("Payment Response : ",
						log.Data{keys.PaymentResponse: paymentResponse, keys.StatusCode: statusCode})

					if paymentResponse.IsReconcilable() {

						//Create GetPayment payment URL
						getPaymentDetailsURL := svc.PaymentsAPIURL + "/private/payments/" + pp.ResourceURI + "/payment-details"
						log.Info("Payment Details URL : " + getPaymentDetailsURL)

						//Call GetPayment payment details from payments API
						paymentDetails, statusCode, err := svc.Payments.GetPaymentDetails(getPaymentDetailsURL, svc.Client, svc.APIKey)

						if err != nil {
							log.Error(err, log.Data{keys.Offset: message.Offset})
							_ = svc.HandleError(err, message.Offset, &paymentDetails)
						}
						log.Info("Payment Details Response : ",
							log.Data{keys.PaymentDetails: paymentDetails, keys.StatusCode: statusCode})

						if isRefundTransaction(pp) {
							svc.handleRefundTransaction(paymentResponse, message, pp)
						} else if paymentDetails.PaymentStatus == "accepted" {

							// We need to remove sensitive data fields for secure applications.
							svc.MaskSensitiveFields(&paymentResponse)

							// Get Eshu resources
							eshus := svc.getEshuResources(message, paymentResponse, paymentDetails, pp.ResourceURI)

							//Add Eshu objects to the Database
							svc.saveEshuResources(message, eshus)

							//Build Payment Transaction database objects
							txns := svc.getTransactionResources(message, paymentResponse, paymentDetails, pp.ResourceURI)

							//Add Payment Transactions to the Database
							svc.saveTransactionResources(message, txns)
						}
					}
				}
			}

		case err = <-svc.Consumer.Errors():
			log.Error(err, log.Data{keys.Topic: svc.Topic})
		}
	}

	// We only get here if we're an error consumer and we've reached out stop offset
	// We will not consume any further messages, so disconnect consumer.
	svc.Shutdown(svc.Topic)

	// The app must not exit until explicitly asked to. If it did, when in
	// a managed environment such as Mesos/Marathon, the app will get
	// restarted and will go on to consume further messages in the error
	// topic and chasing it's own tail, if something is really broken.
	if running {
		select {
		case <-c: // Just wait for a shutdown event
			log.Info("Received close notification")
		}
	}

	wg.Done()

	log.Info("Service successfully shutdown", log.Data{keys.Topic: svc.Topic})
}

// We need a function to mask potentially sensitive data fields in the event it's a secure application.
// Currently there are product types/codes registered against these applications.
func (svc *Service) MaskSensitiveFields(payment *data.PaymentResponse) {
	log.Info("Blanking sensitive fields for secure applications. ")

	// Define the value to be used for masked fields.
	const maskedValue string = ""

	// Find the product code associated with this product type.
	productCode := svc.ProductMap.Codes[payment.Costs[0].ProductType]

	if productCode == 16800 {
		payment.CompanyNumber = maskedValue
		payment.CreatedBy.Email = maskedValue
	}

}

//Shutdown closes all producers and consumers for this service
func (svc *Service) Shutdown(topic string) {

	log.Info("Shutting down service ", log.Data{keys.Topic: topic})

	log.Info("Closing producer", log.Data{keys.Topic: topic})
	err := svc.Producer.Close()
	if err != nil {
		log.Error(fmt.Errorf("error closing producer: %s", err))
	}
	log.Info("Producer successfully closed", log.Data{keys.Topic: svc.Topic})

	log.Info("Closing consumer", log.Data{keys.Topic: topic})
	err = svc.Consumer.Close()
	if err != nil {
		log.Error(fmt.Errorf("error closing consumer: %s", err))
	}
	log.Info("Consumer successfully closed", log.Data{keys.Topic: svc.Topic})
}

// Creates Eshu resources
func (svc *Service) getEshuResources(
	message *sarama.ConsumerMessage,
	paymentResponse data.PaymentResponse,
	paymentDetailsResponse data.PaymentDetailsResponse,
	paymentId string) []models.EshuResourceDao {

	eshus, err := svc.Transformer.GetEshuResources(paymentResponse, paymentDetailsResponse, paymentId)
	if err != nil {
		log.Error(err, log.Data{keys.Offset: message.Offset})
		_ = svc.HandleError(err, message.Offset, &paymentDetailsResponse)
	}
	return eshus
}

// Saves Eshu resources to the Database
func (svc *Service) saveEshuResources(message *sarama.ConsumerMessage, eshus []models.EshuResourceDao) {

	for _, eshu := range eshus {
		err := svc.DAO.CreateEshuResource(&eshu)
		if err != nil {
			log.Error(err, log.Data{keys.Message: "failed to create eshu request in database",
				"data": eshu})
			_ = svc.HandleError(err, message.Offset, &eshus)
		}
	}
}

// Creates Payment Transaction database objects
func (svc *Service) getTransactionResources(
	message *sarama.ConsumerMessage,
	paymentResponse data.PaymentResponse,
	paymentDetailsResponse data.PaymentDetailsResponse,
	paymentId string) []models.PaymentTransactionsResourceDao {

	txns, err := svc.Transformer.GetTransactionResources(paymentResponse, paymentDetailsResponse, paymentId)
	if err != nil {
		log.Error(err, log.Data{keys.Offset: message.Offset})
		_ = svc.HandleError(err, message.Offset, &paymentDetailsResponse)
	}
	return txns
}

// Saves Transaction resources to the database
func (svc *Service) saveTransactionResources(
	message *sarama.ConsumerMessage,
	txns []models.PaymentTransactionsResourceDao) {

	for _, txn := range txns {
		err := svc.DAO.CreatePaymentTransactionsResource(&txn)
		if err != nil {
			log.Error(err, log.Data{keys.Message: "failed to create production request in database",
				"data": txn})
			_ = svc.HandleError(err, message.Offset, &txns)
		}
	}
}

// Creates Refund resources
func (svc *Service) getRefundResource(
	message *sarama.ConsumerMessage,
	paymentResponse data.PaymentResponse,
	refund data.RefundResource,
	paymentId string) models.RefundResourceDao {

	refundResource, err := svc.Transformer.GetRefundResource(paymentResponse, refund, paymentId)
	if err != nil {
		log.Error(err, log.Data{keys.Offset: message.Offset})
		_ = svc.HandleError(err, message.Offset, &refund)
	}
	return refundResource
}

// Saves Refund resources to the database
func (svc *Service) saveRefundResource(
	message *sarama.ConsumerMessage,
	refund models.RefundResourceDao) {

	err := svc.DAO.CreateRefundResource(&refund)
	if err != nil {
		log.Error(err, log.Data{keys.Message: "failed to create refund request in database",
			"data": refund})
		_ = svc.HandleError(err, message.Offset, &refund)
	}
}

func isRefundTransaction(pp data.PaymentProcessed) bool {
	return pp.RefundId != ""
}

func (svc *Service) handleRefundTransaction(paymentResponse data.PaymentResponse, message *sarama.ConsumerMessage, pp data.PaymentProcessed) {
	refund, err := getRefund(paymentResponse, pp)

	if err != nil {
		log.Error(err, log.Data{keys.Message: "Failed to handle refund transaction",
			"data": paymentResponse})
		_ = svc.HandleError(err, message.Offset, &paymentResponse)
	}

	if refund != nil {
		if refund.Status == "success" {
			// We need to remove sensitive data fields for secure applications.
			svc.MaskSensitiveFields(&paymentResponse)

			refundResource := svc.getRefundResource(message, paymentResponse, *refund, pp.ResourceURI)

			svc.saveRefundResource(message, refundResource)
		} else {
			log.Info("Refund found but is not successful. Skipping reconciliation ", log.Data{"Refund": refund})
		}
	}
}

func getRefund(paymentResponse data.PaymentResponse, pp data.PaymentProcessed) (*data.RefundResource, error) {
	for _, ref := range paymentResponse.Refunds {
		if ref.RefundId == pp.RefundId {
			return &ref, nil
		}
	}
	return nil, errors.New("refund id not found in payment refunds")
}
