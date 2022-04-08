package application

import (
	"context"
	"github.com/Diez37/go-skeleton/infrastructure/repository"
	"github.com/diez37/go-packages/repeater"
	"go.opentelemetry.io/otel/trace"
	"time"
)

type Clear interface {
	repeater.Process
}

type clear struct {
	repository repository.Blocker
	tracer     trace.Tracer
}

func NewClear(repository repository.Blocker, tracer trace.Tracer) Clear {
	return &clear{repository: repository, tracer: tracer}
}

func (service *clear) Process(ctx context.Context) error {
	ctx, span := service.tracer.Start(ctx, "service.clear.process")
	defer span.End()

	return service.repository.BlockByDate(ctx, time.Now().In(time.UTC))
}
