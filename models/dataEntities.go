package models

import "time"

// EshuResourceDao represents the Eshu data structure
type EshuResourceDao struct {
	PaymentRef      string    `bson:"payment_reference"`
	ProductCode     int       `bson:"product_code"`
	CompanyNumber   string    `bson:"company_number"`
	FilingDate      string    `bson:"filing_date"`
	MadeUpdate      string    `bson:"made_up_date"`
	TransactionDate time.Time `bson:"transaction_date"`
}

// PaymentTransactionsResourceDao represents the payment transaction data structure
type PaymentTransactionsResourceDao struct {
	TransactionID     string    `bson:"transaction_id"`
	TransactionDate   time.Time `bson:"transaction_date"`
	Email             string    `bson:"email"`
	PaymentMethod     string    `bson:"payment_method"`
	Amount            string    `bson:"amount"`
	CompanyNumber     string    `bson:"company_number"`
	TransactionType   string    `bson:"transaction_type"`
	OrderReference    string    `bson:"order_reference"`
	Status            string    `bson:"status"`
	UserID            string    `bson:"user_id"`
	OriginalReference string    `bson:"original_reference"`
	DisputeDetails    string    `bson:"dispute_details"`
}
