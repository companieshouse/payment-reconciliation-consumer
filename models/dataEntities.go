package models

// EshuResourceDao represents the Eshu data structure
type EshuResourceDao struct {
    PaymentRef    string             `bson:"paymentRef"`
    ProductCode   int                `bson:"productCode"`
    CompanyNumber string             `bson:"companyNumber"`
    FilingDate    string             `bson:"filingDate"`
    MadeUpdate    string             `bson:"madeUpdate"`
}

// PaymentTransactionsResourceDao represents the payment transaction data structure
type PaymentTransactionsResourceDao struct {
    TransactionID     string             `bson:"transactionID"`
    TransactionDate   string             `bson:"transactionDate"`
    Email             string             `bson:"email"`
    PaymentMethod     string             `bson:"paymentMethod"`
    Amount            string             `bson:"amount"`
    CompanyNumber     string             `bson:"companyNumber"`
    TransactionType   string             `bson:"transactionType"`
    OrderReference    string             `bson:"orderReference"`
    Status            string             `bson:"status"`
    UserID            string             `bson:"userID"`
    OriginalReference string             `bson:"originalReference"`
    DisputeDetails    string             `bson:"disputeDetails"`
}
