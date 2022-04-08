package http

import (
	"context"
	"github.com/Diez37/go-skeleton/application"
	"github.com/Diez37/go-skeleton/infrastructure/config"
	"github.com/Diez37/go-skeleton/interface/http/api"
	"github.com/diez37/go-packages/container"
	"github.com/diez37/go-packages/log"
	httpServer "github.com/diez37/go-packages/server/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"net"
	"net/http"
)

// Serve configuration and running http server
func Serve(ctx context.Context, container container.Container, logger log.Logger, service application.Token, tracer trace.Tracer) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	errGroup := &errgroup.Group{}

	err := container.Invoke(func(
		server *http.Server,
		config *httpServer.Config,
		tokenConfig *config.Token,
		validator *validator.Validate,
		router chi.Router,
	) {
		logger.Info("http server: add '/token' handler")
		router.Mount("/token", api.Router(logger, tokenConfig, service, validator, tracer))

		errGroup.Go(func() error {
			defer cancelFunc()

			logger.Infof("http server: started")

			server.BaseContext = func(_ net.Listener) context.Context {
				return ctx
			}

			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				return err
			}

			return nil
		})

		errGroup.Go(func() error {
			<-ctx.Done()

			logger.Infof("http server: shutdown")

			return server.Close()
		})
	})

	if err != nil {
		return err
	}

	return errGroup.Wait()
}
