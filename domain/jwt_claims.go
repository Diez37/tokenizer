package domain

import (
	"github.com/google/uuid"
	"time"
)

type JwtClaims struct {
	Login     uuid.UUID
	ExpiresIn time.Time
}
