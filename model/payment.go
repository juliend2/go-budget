package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Payment struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ExpenseID primitive.ObjectID `bson:"expense_id" json:"expense_id"`
	Amount    int                `bson:"amount" json:"amount"`
	PaidAt    time.Time          `bson:"paid_at" json:"paid_at"`
}
