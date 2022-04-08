package repository

import (
	"github.com/google/uuid"
	"time"
)

type RefreshToken struct {
	UUID        uuid.UUID `db:"uuid"`
	Login       uuid.UUID `db:"login"`
	Ip          string    `db:"ip"`
	Fingerprint string    `db:"fingerprint"`
	UserAgent   string    `db:"user_agent"`
	CreatedAt   time.Time `db:"created_at"`
	ExpiresIn   time.Time `db:"expires_in"`
}
