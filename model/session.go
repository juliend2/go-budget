package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Session struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SecureID  string             `bson:"secure_id" json:"secure_id"`
	Email     string             `bson:"email" json:"email"`
	Name      string             `bson:"name" json:"name"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
}
