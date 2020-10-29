package data

// PaymentProcessed represents payment change avro schema
type PaymentProcessed struct {
	ResourceURI string `avro:"payment_resource_id"`
	RefundId    string `avro:"refund_id,omitempty"`
}
