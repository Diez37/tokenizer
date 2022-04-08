package v1

import (
	"github.com/Diez37/go-skeleton/application"
	"github.com/Diez37/go-skeleton/infrastructure/config"
	"github.com/diez37/go-packages/log"
	"github.com/diez37/go-packages/router/middlewares"
	"github.com/go-chi/chi/v5"
	"github.com/go-http-utils/headers"
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

	api := NewApi(config, logger, service, validator, tracer)

	ipMiddleware := middlewares.NewIP(middlewares.IpWithName(IpFieldName)).Middleware
	userAgentMiddleware := middlewares.NewString(
		logger,
		middlewares.WithHeader(headers.UserAgent),
		middlewares.WithName(UserAgentFieldName),
	).Middleware
	refreshTokenMiddleware := middlewares.NewUUID(
		logger,
		middlewares.WithCookie(RefreshTokenFieldName),
		middlewares.WithName(RefreshTokenFieldName),
	).Middleware

	router.Route("/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(ipMiddleware)
			r.Use(userAgentMiddleware)
			r.Put("/", api.Create)
		})

		r.Group(func(r chi.Router) {
			r.Use(BearerAuthorization)
			r.Get("/", api.Read)
			r.Options("/", api.Validation)

			r.Group(func(r chi.Router) {
				r.Use(refreshTokenMiddleware)
				r.Delete("/all", api.DeleteAll)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(refreshTokenMiddleware)

			r.Group(func(r chi.Router) {
				r.Use(ipMiddleware)
				r.Use(userAgentMiddleware)
				r.Post("/", api.Update)
			})

			r.Delete("/", api.Delete)
		})
	})

	return router
}
