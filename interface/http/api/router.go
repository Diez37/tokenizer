package api

import (
	"github.com/Diez37/go-skeleton/application"
	"github.com/Diez37/go-skeleton/infrastructure/config"
	v1 "github.com/Diez37/go-skeleton/interface/http/api/v1"
	"github.com/diez37/go-packages/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel/trace"
)

func Router(
	logger log.Logger,
	config *config.Token,
	service application.Token,
	validator *validator.Validate,
	tracer trace.Tracer,
) chi.Router {
	router := chi.NewRouter()

	router.Mount("/api", v1.Router(logger, config, service, validator, tracer))

	return router
}
