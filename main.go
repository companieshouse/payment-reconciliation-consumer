package main

import (
	"fmt"
	gologger "log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	"github.com/companieshouse/chs.go/kafka/resilience"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payment-reconciliation-consumer/config"
	"github.com/companieshouse/payment-reconciliation-consumer/handlers"
	"github.com/companieshouse/payment-reconciliation-consumer/service"
	"github.com/gorilla/pat"
)

func main() {
	log.Namespace = "payment-reconciliation-consumer"

	// Push the Sarama logs into our custom writer
	sarama.Logger = gologger.New(&log.Writer{}, "[Sarama] ", gologger.LstdFlags)

	cfg, err := config.Get()
	if err != nil {
		log.Error(fmt.Errorf("error configuring service: %s. Exiting", err), nil)
		return
	}

	log.Info("intialising payment-reconciliation-consumer service...")

	mainChannel := make(chan os.Signal, 1)
	retryChannel := make(chan os.Signal, 1)

	svc, err := service.New(cfg.PaymentProcessedTopic, cfg.PaymentReconciliationGroupName, cfg, nil)
	if err != nil {
		log.Error(fmt.Errorf("error initialising main consumer service: '%s'. Exiting", err), nil)
		return
	}

	var wg sync.WaitGroup
	if !cfg.IsErrorConsumer {
		retrySvc, err := getRetryService(cfg)
		if err != nil {
			log.Error(fmt.Errorf("error initialising retry consumer service: '%s'. Exiting", err), nil)
			svc.Shutdown(cfg.PaymentProcessedTopic)
			return
		}
		wg.Add(1)
		go retrySvc.Start(&wg, retryChannel)
	}

	wg.Add(1)
	go svc.Start(&wg, mainChannel)

	router := pat.New()
	handlers.Init(router)
	go func() {
		log.Info("Starting HTTP server on :" + "8080")
		if err := http.ListenAndServe(":8080", router); err != nil {
			log.Error(fmt.Errorf("error starting HTTP server: %s", err), nil)
		}
	}()

	waitForServiceClose(&wg, mainChannel, retryChannel)

	log.Info("Application successfully shutdown")

}

func getRetryService(cfg *config.Config) (*service.Service, error) {

	retry := &resilience.ServiceRetry{
		ThrottleRate: time.Duration(cfg.RetryThrottleRate),
		MaxRetries:   cfg.MaxRetryAttempts,
	}

	retrySvc, err := service.New(cfg.PaymentProcessedTopic, cfg.PaymentReconciliationGroupName, cfg, retry)
	if err != nil {
		log.Error(fmt.Errorf("error initialising retry consumer service: %s", err), nil)
		return nil, err
	}

	return retrySvc, nil
}

// waitForServiceClose will receive the close signal and forward a notification
// to all service (go routines) to ensure that they clean up (for example their
// consumers and producers) and exit gracefully.
func waitForServiceClose(wg *sync.WaitGroup, mainChannel, retryChannel chan os.Signal) {

	// Channel to fan-out interrupt/kill notifications
	notificationChannel := make(chan os.Signal, 1)
	signal.Notify(notificationChannel, os.Interrupt, os.Kill, syscall.SIGTERM)

	select {
	case notification := <-notificationChannel:
		// Falls into this block to successfully close consumer after service shutdown
		log.Info("Close signal received, fanning out...")
		log.Debug("Sending notification to main consumer channel")
		mainChannel <- notification

		log.Debug("Sending notification to retry consumer channel")
		retryChannel <- notification

		log.Info("Fan out completed")
	}
	wg.Wait()
}
