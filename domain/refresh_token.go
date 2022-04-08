package domain

import (
	"github.com/google/uuid"
	"net"
)

type RefreshToken struct {
	UUID        uuid.UUID
	Login       uuid.UUID
	Ip          net.IP
	Fingerprint string
	UserAgent   string
}
