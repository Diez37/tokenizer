package application

import (
	"context"
	"github.com/Diez37/go-skeleton/infrastructure/repository"
	"github.com/diez37/go-packages/clients/db"
	"github.com/diez37/go-packages/repeater"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"sync"
)

const (
	saverInitCap = 500
)

type Saver interface {
	repeater.Process
	repository.Finder
	repository.Saver
}

type saver struct {
	rwMutex *sync.RWMutex

	repository repository.Repository

	models        []*repository.RefreshToken
	modelsByLogin map[uuid.UUID][]*repository.RefreshToken
	modelsByUUID  map[uuid.UUID]*repository.RefreshToken

	tracer trace.Tracer
}

func NewSaver(tokenRepository repository.Repository, tracer trace.Tracer) Saver {
	return &saver{
		rwMutex:       &sync.RWMutex{},
		repository:    tokenRepository,
		models:        make([]*repository.RefreshToken, 0, saverInitCap),
		modelsByLogin: map[uuid.UUID][]*repository.RefreshToken{},
		modelsByUUID:  map[uuid.UUID]*repository.RefreshToken{},
		tracer:        tracer,
	}
}

func (service *saver) Process(ctx context.Context) error {
	ctx, span := service.tracer.Start(ctx, "service.saver.process")
	defer span.End()

	service.rwMutex.Lock()
	defer service.rwMutex.Unlock()

	if len(service.models) == 0 {
		return nil
	}

	err := service.repository.Insert(ctx, service.models...)
	if err == nil {
		service.models = make([]*repository.RefreshToken, 0, saverInitCap)
		service.modelsByLogin = map[uuid.UUID][]*repository.RefreshToken{}
		service.modelsByUUID = map[uuid.UUID]*repository.RefreshToken{}
	}

	return err
}

func (service *saver) FindByLogin(ctx context.Context, login uuid.UUID) ([]*repository.RefreshToken, error) {
	ctx, span := service.tracer.Start(ctx, "finder.login")
	defer span.End()

	span.SetAttributes(
		attribute.String("login", login.String()),
		attribute.String("repository", "service"),
		attribute.String("service", "saver"),
	)

	tokens, err := service.repository.FindByLogin(ctx, login)
	if err != nil && err != db.RecordNotFoundError {
		return nil, err
	}

	service.rwMutex.RLock()
	defer service.rwMutex.RUnlock()

	if tokensByLogin, exist := service.modelsByLogin[login]; exist {
		tokens = append(tokens, tokensByLogin...)
	}

	if len(tokens) == 0 {
		return nil, db.RecordNotFoundError
	}

	return tokens, nil
}

func (service *saver) FindByUUID(ctx context.Context, uuid uuid.UUID) (*repository.RefreshToken, error) {
	ctx, span := service.tracer.Start(ctx, "finder.uuid")
	defer span.End()

	span.SetAttributes(
		attribute.String("uuid", uuid.String()),
		attribute.String("repository", "service"),
		attribute.String("service", "saver"),
	)

	token, err := service.repository.FindByUUID(ctx, uuid)
	if err != nil && err != db.RecordNotFoundError {
		return nil, err
	}

	if token != nil {
		return token, nil
	}

	service.rwMutex.RLock()
	defer service.rwMutex.RUnlock()

	if token, exist := service.modelsByUUID[uuid]; exist {
		return token, nil
	}

	return nil, db.RecordNotFoundError
}

func (service *saver) Insert(ctx context.Context, tokens ...*repository.RefreshToken) error {
	ctx, span := service.tracer.Start(ctx, "saver.insert")
	defer span.End()

	span.SetAttributes(
		attribute.Int("length", len(tokens)),
		attribute.String("repository", "service"),
		attribute.String("service", "saver"),
	)

	service.rwMutex.Lock()
	defer service.rwMutex.Unlock()

	service.models = append(service.models, tokens...)

	for _, token := range tokens {
		if _, exist := service.modelsByLogin[token.Login]; !exist {
			service.modelsByLogin[token.Login] = []*repository.RefreshToken{}
		}

		service.modelsByLogin[token.Login] = append(service.modelsByLogin[token.Login], token)
		service.modelsByUUID[token.UUID] = token
	}

	return nil
}
