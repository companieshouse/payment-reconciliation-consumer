package data

// PaymentDetailsResponse represents a response from the payment details service GET payment endpoint
type PaymentDetailsResponse struct {
    CardType        string `json:"card_type"`
    PaymentID       string `json:"payment_id"`
    TransactionDate string `json:"transaction_date"`
    PaymentStatus   string `json:"payment_status"`
}
