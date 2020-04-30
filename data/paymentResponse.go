package data

import (
	"time"
)

const DataMaintenance = "data-maintenance"
const OrderableItem = "orderable-item"
const Penalty = "penalty"

// PaymentResponse represents a response from the payment service GET payment endpoint
type PaymentResponse struct {
	Amount                  string       `json:"amount"`
	AvailablePaymentMethods []string     `json:"available_payment_methods,omitempty"`
	CompletedAt             time.Time    `json:"completed_at,omitempty"`
	CreatedAt               time.Time    `json:"created_at,omitempty"`
	CreatedBy               Created      `json:"created_by"`
	Description             string       `json:"description"`
	Links                   PaymentLinks `json:"links"`
	PaymentMethod           string       `json:"payment_method,omitempty"`
	Reference               string       `json:"reference,omitempty"`
	CompanyNumber           string       `json:"company_number,omitempty"`
	Status                  string       `json:"status"`
	Costs                   []Cost       `json:"costs"`
	Etag                    string       `json:"etag"`
	Kind                    string       `json:"kind"`
}

// Cost represents a cost data structure
type Cost struct {
	Amount                  string            `json:"amount"`
	AvailablePaymentMethods []string          `json:"available_payment_methods"`
	ClassOfPayment          []string          `json:"class_of_payment"`
	Description             string            `json:"description"`
	DescriptionIdentifier   string            `json:"description_identifier"`
	ProductType             string            `json:"product_type"`
	DescriptionValues       map[string]string `json:"description_values"`
}

// PaymentLinks is a set of URLs related to the resource, including self
type PaymentLinks struct {
	Journey  string `json:"journey"`
	Resource string `json:"resource"`
	Self     string `json:"self" validate:"required"`
}

// Created data
type Created struct {
	Email    string `json:"email"`
	Forename string `json:"forename"`
	ID       string `json:"id"`
	Surname  string `json:"surname"`
}

// Indicates whether the payment is reconcilable or not.
func (payment PaymentResponse) IsReconcilable() bool {
	return (payment.Costs[0].ClassOfPayment[0] == DataMaintenance ||
		payment.Costs[0].ClassOfPayment[0] == OrderableItem)
}
