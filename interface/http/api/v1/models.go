package v1

import (
	"github.com/google/uuid"
	"time"
)

type Create struct {
	Login       uuid.UUID `json:"login" validate:"required"`
	Fingerprint string    `json:"fingerprint" validate:"required"`
}

type RefreshToken struct {
	Fingerprint string `json:"fingerprint" validate:"required"`
}

type AccessToken struct {
	Login     uuid.UUID `json:"login"`
	ExpiresIn time.Time `json:"expires_in"`
}
