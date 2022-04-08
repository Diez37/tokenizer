package repository

import (
	"context"
	"github.com/google/uuid"
	"time"
)

type Finder interface {
	FindByLogin(ctx context.Context, login uuid.UUID) ([]*RefreshToken, error)
	FindByUUID(ctx context.Context, uuid uuid.UUID) (*RefreshToken, error)
}

type Saver interface {
	Insert(ctx context.Context, tokens ...*RefreshToken) error
}

type Blocker interface {
	BlockByUUID(ctx context.Context, uuids ...uuid.UUID) error
	BlockByDate(ctx context.Context, date time.Time) error
}

type Repository interface {
	Finder
	Saver
	Blocker
}
