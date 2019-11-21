package service

import (
	"fmt"
	"github.com/companieshouse/payment-reconciliation-consumer/dao"
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
	log.Info("Successfully received schema", log.Data{"schema_name": schemaName})

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

	log.Info("Start Request Create resilient Kafka service", log.Data{"base_topic": consumerTopic, "app_name": appName, "maxRetries": maxRetries, "producer": p})
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
			log.Error(err, log.Data{"topic": topicName})
		}
		log.Info("error queue consumer will stop when backlog offset reached", log.Data{"backlog_offset": stopAtOffset})
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
			log.Trace("Committing message", log.Data{"offset": message.Offset})
			svc.Consumer.MarkOffset(message, "")
			if err := svc.Consumer.CommitOffsets(); err != nil {
				log.Error(err, log.Data{"offset": message.Offset})
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
						log.Error(err, log.Data{"message_offset": message.Offset})
						svc.HandleError(err, message.Offset, &message.Value)
						continue
					}

					//Create GetPayment payment session URL
					getPaymentURL := svc.PaymentsAPIURL + "/payments/" + pp.ResourceURI
					log.Info("Payment URL : " + getPaymentURL)

					//Call GetPayment payment session from payments API
					paymentResponse, statusCode, err := svc.Payments.GetPayment(getPaymentURL, svc.Client, svc.APIKey)
					if err != nil {
						log.Error(err, log.Data{"message_offset": message.Offset})
						svc.HandleError(err, message.Offset, &paymentResponse)
					}
					log.Info("Payment Response : ", log.Data{"payment_response": paymentResponse, "status_code": statusCode})

					if paymentResponse.Costs[0].ClassOfPayment[0] == "data-maintenance" {

						//Create GetPayment payment URL
						getPaymentDetailsURL := svc.PaymentsAPIURL + "/private/payments/" + pp.ResourceURI + "/payment-details"
						log.Info("Payment Details URL : " + getPaymentDetailsURL)

						//Call GetPayment payment details from payments API
						paymentDetails, statusCode, err := svc.Payments.GetPaymentDetails(getPaymentDetailsURL, svc.Client, svc.APIKey)

						if err != nil {
							log.Error(err, log.Data{"message_offset": message.Offset})
							svc.HandleError(err, message.Offset, &paymentDetails)
						}
						log.Info("Payment Details Response : ", log.Data{"payment_details": paymentDetails, "status_code": statusCode})

						//Filter accepted payments from GovPay
						if paymentDetails.PaymentStatus == "accepted" {

							// We need to remove sensitive data fields for secure applications.
							svc.MaskSensitiveFields(&paymentResponse)

							//Get Eshu resource
							eshu, err := svc.Transformer.GetEshuResource(paymentResponse, paymentDetails, pp.ResourceURI)
							if err != nil {
								log.Error(err, log.Data{"message_offset": message.Offset})
								svc.HandleError(err, message.Offset, &paymentDetails)
							}

							//Add Eshu object to the Database
							err = svc.DAO.CreateEshuResource(&eshu)
							if err != nil {
								log.Error(err, log.Data{"message": "failed to create eshu request in database",
									"data": eshu})
								svc.HandleError(err, message.Offset, &eshu)
							}

							//Build Payment Transaction database object
							payTrans, err := svc.Transformer.GetTransactionResource(paymentResponse, paymentDetails, pp.ResourceURI)
							if err != nil {
								log.Error(err, log.Data{"message_offset": message.Offset})
								svc.HandleError(err, message.Offset, &paymentDetails)
							}

							//Add Payment Transaction to the Database
							err = svc.DAO.CreatePaymentTransactionsResource(&payTrans)
							if err != nil {
								log.Error(err, log.Data{"message": "failed to create production request in database",
									"data": payTrans})
								svc.HandleError(err, message.Offset, &payTrans)
							}
						}
					}
				}
			}

		case err = <-svc.Consumer.Errors():
			log.Error(err, log.Data{"topic": svc.Topic})
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

	log.Info("Service successfully shutdown", log.Data{"topic": svc.Topic})
}

// We need a function to mask potentially sensitive data fields in the event it' a secure application.
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

	log.Info("Shutting down service ", log.Data{"topic": topic})

	log.Info("Closing producer", log.Data{"topic": topic})
	err := svc.Producer.Close()
	if err != nil {
		log.Error(fmt.Errorf("error closing producer: %s", err))
	}
	log.Info("Producer successfully closed", log.Data{"topic": svc.Topic})

	log.Info("Closing consumer", log.Data{"topic": topic})
	err = svc.Consumer.Close()
	if err != nil {
		log.Error(fmt.Errorf("error closing consumer: %s", err))
	}
	log.Info("Consumer successfully closed", log.Data{"topic": svc.Topic})
}
