package handlers

import (
	"github.com/companieshouse/chs.go/log"
	"github.com/gorilla/pat"
)

func Init(r *pat.Router) {
	log.Info("initialising healthcheck endpoint beneath basePath: /payment-reconciliation-consumer")

	appRouter := r.PathPrefix("/payment-reconciliation-consumer").Subrouter()

	appRouter.Path("/healthcheck").Methods("GET").HandlerFunc(HealthCheck)
}