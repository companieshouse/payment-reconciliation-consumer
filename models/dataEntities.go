package models

import (
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type EshuResourceDao struct {
    ID            primitive.ObjectID `bson:"_id"`
    PaymentRef    string             `bson:"paymentRef"`
    ProductCode   int                `bson:"productCode"`
    CompanyNumber string             `bson:"companyNumber"`
    FilingDate    string             `bson:"filingDate"`
    MadeUpdate    string             `bson:"madeUpdate"`
}

type PaymentTransactionsResourceDao struct {
    ID                primitive.ObjectID `bson:"_id"`
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
