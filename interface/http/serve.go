package http

import (
	"context"
	"github.com/diez37/go-packages/container"
	"github.com/diez37/go-packages/log"
	httpServer "github.com/diez37/go-packages/server/http"
	"golang.org/x/sync/errgroup"
	"net/http"
)

// Serve configuration and running http server
func Serve(ctx context.Context, container container.Container, logger log.Logger) error {
	return container.Invoke(func(server *http.Server, config *httpServer.Config) error {
		// TODO: adding paths and handlers to router only here
		errGroup := &errgroup.Group{}

		errGroup.Go(func() error {
			logger.Infof("http server: started")
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				return err
			}

			return nil
		})

		errGroup.Go(func() error {
			<-ctx.Done()

			logger.Infof("http server: shutdown")

			ctxTimeout, cancelFnc := context.WithTimeout(context.Background(), config.ShutdownTimeout)
			defer cancelFnc()

			return server.Shutdown(ctxTimeout)
		})

		return errGroup.Wait()
	})
}
