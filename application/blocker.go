package application

import (
	"context"
	"github.com/Diez37/go-skeleton/infrastructure/repository"
	"github.com/diez37/go-packages/repeater"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"sync"
	"time"
)

const (
	blockerInitCap = 5000
)

type Blocker interface {
	repeater.Process
	repository.Blocker
}

type blocker struct {
	mutex *sync.Mutex

	repository repository.Repository

	uuids  []uuid.UUID
	tracer trace.Tracer
}

func NewBlocker(repository repository.Repository, tracer trace.Tracer) Blocker {
	return &blocker{
		mutex:      &sync.Mutex{},
		repository: repository,
		uuids:      make([]uuid.UUID, 0, blockerInitCap),
		tracer:     tracer,
	}
}

func (service *blocker) Process(ctx context.Context) error {
	ctx, span := service.tracer.Start(ctx, "service.blocker.process")
	defer span.End()

	var errs error
	wg := &sync.WaitGroup{}

	service.mutex.Lock()

	wg.Add(1)
	go func(ctx context.Context, uuids ...uuid.UUID) {
		wg.Done()
		if err := service.repository.BlockByUUID(ctx, uuids...); err != nil {
			errs = multierr.Append(errs, err)

			if err := service.BlockByUUID(ctx, uuids...); err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}(ctx, service.uuids...)

	service.uuids = make([]uuid.UUID, 0, blockerInitCap)

	service.mutex.Unlock()

	wg.Wait()

	return errs
}

func (service *blocker) BlockByUUID(ctx context.Context, uuids ...uuid.UUID) error {
	ctx, span := service.tracer.Start(ctx, "blocker.uuid")
	defer span.End()

	span.SetAttributes(
		attribute.Int("length", len(uuids)),
		attribute.String("repository", "service"),
		attribute.String("service", "blocker"),
	)

	service.mutex.Lock()
	defer service.mutex.Unlock()

	service.uuids = append(service.uuids, uuids...)

	return nil
}

func (service *blocker) BlockByDate(ctx context.Context, date time.Time) error {
	ctx, span := service.tracer.Start(ctx, "blocker.date")
	defer span.End()

	span.SetAttributes(
		attribute.String("repository", "service"),
		attribute.String("service", "blocker"),
	)

	return service.repository.BlockByDate(ctx, date)
}
