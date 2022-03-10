package cli

import (
	"context"
	"github.com/Diez37/go-skeleton/interface/http"
	"github.com/diez37/go-packages/app"
	"github.com/diez37/go-packages/closer"
	"github.com/diez37/go-packages/configurator"
	bindFlags "github.com/diez37/go-packages/configurator/bind_flags"
	"github.com/diez37/go-packages/container"
	"github.com/diez37/go-packages/log"
	"github.com/spf13/cobra"
)

const (
	// AppName name of application
	AppName = "go-skeleton"
)

// NewRootCommand creating, configuration and return cobra.Command for root command
func NewRootCommand() (*cobra.Command, error) {
	container := container.GetContainer()

	cmd := &cobra.Command{
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return container.Invoke(func(generalConfig *app.Config, configurator configurator.Configurator) {
				app.Configuration(generalConfig, configurator, app.WithAppName(AppName))
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return container.Invoke(func(generalConfig *app.Config, logger log.Logger, closer closer.Closer) error {
				logger.Infof("app: %s started", generalConfig.Name)
				logger.Infof("app: pid - %d", generalConfig.PID)

				ctx, cancelFnc := context.WithCancel(closer.GetContext())
				defer cancelFnc()

				return http.Serve(ctx, container, logger)
			})
		},
	}

	cmd, err := bindFlags.CobraCmd(container, cmd,
		bindFlags.HttpServer,
		bindFlags.Logger,
		bindFlags.Tracer,
	)
	if err != nil {
		return nil, err
	}

	return cmd, nil
}
