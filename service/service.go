package service

import (
	"fmt"
	"github.com/companieshouse/payment-reconciliation-consumer/dao"
	"github.com/companieshouse/payment-reconciliation-consumer/models"
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
	ApiKey          string
	PaymentsAPIURL  string
	DAO             dao.Service
	TranCollection  string
	ProdCollection  string
	ProductMap      *config.ProductMap
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

	productMap, err := cfg.GetProductMap()
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

	dao := dao.NewDAOService(cfg)

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
		ApiKey:          cfg.ChsAPIKey,
		PaymentsAPIURL:  cfg.PaymentsAPIURL,
		DAO:             dao,
		TranCollection:  cfg.TransactionsCollection,
		ProdCollection:  cfg.ProductsCollection,
		ProductMap:      productMap,
	}, nil

}

// Start begins the service - Messages are consumed from the payment-processed
// topic
func (svc *Service) Start(wg *sync.WaitGroup, c chan os.Signal) {
	log.Info("service starting, consuming from " + svc.Topic + " topic")

	// If we're an error consumer, then capture the tail of the topic, and only consume up to that offset.
	stopAtOffset := int64(-1)
	var err error
	if svc.IsErrorConsumer {
		stopAtOffset, err = client.TopicOffset(svc.BrokerAddr, svc.Topic)
		if err != nil {
			log.Error(err, log.Data{"topic": svc.Topic})
		}
		log.Info("error queue consumer will stop when backlog offset reached", log.Data{"backlog_offset": stopAtOffset})
	}

	var message *sarama.ConsumerMessage

	// We want to stop the processing of the service if consuming from an
	// error queue if all messages that were initially in the queue have
	// been cleared using the stopAtOffset
	running := true
	for running && (stopAtOffset == -1 || message == nil || message.Offset < stopAtOffset) {

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

					// Get the payment session first
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

					//Create Get payment session URL
					getPaymentURL := svc.PaymentsAPIURL + "/payments/" + pp.ResourceURI
					log.Info("Payment URL : " + getPaymentURL)
					//Call Get payment session from payments API
					paymentResponse, err := payment.Get(getPaymentURL, svc.Client, svc.ApiKey)
					if err != nil {
						log.Error(err, log.Data{"message_offset": message.Offset})
						svc.HandleError(err, message.Offset, &paymentResponse)
					}
					log.Info("Payment Response : ", log.Data{"payment_response": paymentResponse})
					//Get Cost from payment session Cost array
					cost, err := paymentResponse.GetCost("cic-report")
					if err != nil {
						log.Error(err, log.Data{"message_offset": message.Offset})
						svc.HandleError(err, message.Offset, &paymentResponse)
					}
					log.Info("Cost : ", log.Data{"cost": cost})
					productCode := svc.ProductMap.Codes[cost.ProductType]

					//Create Get payment URL
					getPaymentDetailsURL := svc.PaymentsAPIURL + "/private/payments/" + pp.ResourceURI + "/payment-details"
					log.Info("Payment Details URL : " + getPaymentDetailsURL)
					//Call Get payment details from payments API
					paymentDetails, err := payment.GetDetails(getPaymentDetailsURL, svc.Client, svc.ApiKey)
					if err != nil {
						log.Error(err, log.Data{"message_offset": message.Offset})
						svc.HandleError(err, message.Offset, &paymentDetails)
					}
					log.Info("Payment Details : ", log.Data{"payment_details": paymentDetails})

					//Build Eshu database object
					eshu := models.EshuResourceDao{
						PaymentRef:    paymentDetails.PaymentID,
						ProductCode:   productCode,
						CompanyNumber: paymentResponse.CompanyNumber,
						FilingDate:    "",
						MadeUpdate:    "",
					}

					//Add Eshu object to the Database
					err = svc.DAO.CreateEshuResource(&eshu, svc.TranCollection)
					if err != nil {
						log.Error(err, log.Data{"message": "failed to create eshu request in database",
							"data":       eshu,
							"collection": svc.TranCollection})
						svc.HandleError(err, message.Offset, &eshu)
					}

					//Build Payment Transaction database object
					payTrans := models.PaymentTransactionsResourceDao{
						TransactionID:     paymentDetails.PaymentID,
						TransactionDate:   paymentDetails.TransactionDate,
						Email:             paymentResponse.CreatedBy.Email,
						PaymentMethod:     paymentResponse.PaymentMethod,
						Amount:            paymentResponse.Amount,
						CompanyNumber:     paymentResponse.CompanyNumber,
						TransactionType:   "Immediate bill",
						OrderReference:    paymentResponse.Reference,
						Status:            paymentDetails.PaymentStatus,
						UserID:            "system",
						OriginalReference: "",
						DisputeDetails:    ""}

					//Add Payment Transaction to the Database
					err = svc.DAO.CreatePaymentTransactionsResource(&payTrans, svc.ProdCollection)
					if err != nil {
						log.Error(err, log.Data{"message": "failed to create production request in database",
							"data":       payTrans,
							"collection": svc.ProdCollection})
						svc.HandleError(err, message.Offset, &payTrans)
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
