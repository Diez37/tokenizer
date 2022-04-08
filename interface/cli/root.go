package cli

import (
	"context"
	"fmt"
	"github.com/Diez37/go-skeleton/application"
	"github.com/Diez37/go-skeleton/infrastructure/config"
	container2 "github.com/Diez37/go-skeleton/infrastructure/container"
	"github.com/Diez37/go-skeleton/infrastructure/repository"
	"github.com/Diez37/go-skeleton/interface/http"
	"github.com/diez37/go-packages/app"
	"github.com/diez37/go-packages/closer"
	"github.com/diez37/go-packages/configurator"
	bindFlags "github.com/diez37/go-packages/configurator/bind_flags"
	"github.com/diez37/go-packages/container"
	"github.com/diez37/go-packages/log"
	"github.com/diez37/go-packages/repeater"
	"github.com/golang-jwt/jwt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"strings"
	"sync"
	"time"
)

const (
	// AppName name of application
	AppName = "tokenizer"
)

// NewRootCommand creating, configuration and return cobra.Command for root command
func NewRootCommand() (*cobra.Command, error) {
	container := container.GetContainer()

	if err := container2.AddProvide(container); err != nil {
		return nil, err
	}

	cmd := &cobra.Command{
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return container.Invoke(func(generalConfig *app.Config, configurator configurator.Configurator) {
				app.Configuration(generalConfig, configurator, app.WithAppName(AppName))
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return container.Invoke(func(
				generalConfig *app.Config,
				logger log.Logger,
				closer closer.Closer,
				migrator *migrate.Migrate,
				repository repository.Repository,
				tokenConfig *config.Token,
				repeatService repeater.Repeater,
				tracer trace.Tracer,
			) error {
				logger.Infof("app: %s started", generalConfig.Name)
				logger.Infof("app: pid - %d", generalConfig.PID)

				if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
					return err
				}

				ctx, cancelFunc := context.WithCancel(closer.GetContext())
				defer cancelFunc()

				var errs error
				wg := &sync.WaitGroup{}
				mutex := &sync.Mutex{}

				blocker := application.NewBlocker(repository, tracer)
				saver := application.NewSaver(repository, tracer)
				clear := application.NewClear(repository, tracer)

				jwt.TimeFunc = func() time.Time {
					return time.Now().In(time.UTC)
				}

				wg.Add(1)
				go func() {
					defer wg.Done()

					err := http.Serve(
						ctx,
						container,
						logger,
						application.NewToken(tokenConfig, logger, saver, saver, blocker, tracer),
						tracer,
					)
					if err != nil {
						defer cancelFunc()

						mutex.Lock()
						defer mutex.Unlock()

						errs = multierr.Append(errs, errs)
					}
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()

					repeatService.
						AddProcess("blocker", tokenConfig.DelayBlocker, blocker).
						AddProcess("saver", tokenConfig.DelaySaver, saver).
						AddProcess("clear", tokenConfig.DelayClear, clear).
						Serve(ctx)

					ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute)
					defer cancelFunc()

					if err := blocker.Process(ctx); err != nil {
						mutex.Lock()
						defer mutex.Unlock()

						errs = multierr.Append(errs, errs)
					}

					if err := saver.Process(ctx); err != nil {
						mutex.Lock()
						defer mutex.Unlock()

						errs = multierr.Append(errs, errs)
					}

					if err := clear.Process(ctx); err != nil {
						mutex.Lock()
						defer mutex.Unlock()

						errs = multierr.Append(errs, errs)
					}
				}()

				wg.Wait()
				return errs
			})
		},
	}

	cmd, err := bindFlags.CobraCmd(container, cmd,
		bindFlags.HttpServer,
		bindFlags.Logger,
		bindFlags.Tracer,
		bindFlags.DataBase,
		bindFlags.Migrator,
	)
	if err != nil {
		return nil, err
	}

	err = container.Invoke(func(tokenConfig *config.Token) {
		cmd.PersistentFlags().StringVar(&tokenConfig.Secret, config.TokensSecretFieldName, config.TokensSecretDefault, "")
		cmd.PersistentFlags().UintVar(&tokenConfig.MaximumTokens, config.TokensMaximumTokensFieldName, config.TokensMaximumTokensDefault, "maximum tokens on one account")
		cmd.PersistentFlags().DurationVar(&tokenConfig.DelayClear, config.TokensDelayClearFieldName, config.TokensDelayClearDefault, "")
		cmd.PersistentFlags().DurationVar(&tokenConfig.AccessLifetime, config.TokensAccessLifetimeFieldName, config.TokensAccessLifetimeDefault, "")
		cmd.PersistentFlags().DurationVar(&tokenConfig.RefreshLifetime, config.TokensRefreshLifetimeFieldName, config.TokensRefreshLifetimeDefault, "")
		cmd.PersistentFlags().DurationVar(&tokenConfig.DelayBlocker, config.TokensDelayBlockerFieldName, config.TokensDelayBlockerDefault, "")
		cmd.PersistentFlags().DurationVar(&tokenConfig.DelaySaver, config.TokensDelaySaverFieldName, config.TokensDelaySaverDefault, "")
		cmd.PersistentFlags().StringSliceVar(&tokenConfig.RefreshCheckFields, config.TokensCheckFieldsForRefreshFieldName, config.TokensCheckFieldsForRefresh, "")
		cmd.PersistentFlags().StringVar(
			&tokenConfig.AccessViolation,
			config.TokensRefreshActionOnAccessViolation,
			config.TokensAccessViolationActionDefault,
			fmt.Sprintf("availably [%s]", strings.Join([]string{
				config.TokensAccessViolationActionDisableAll,
				config.TokensAccessViolationActionDisableCurrent,
				config.TokensAccessViolationActionNone,
			}, ",")),
		)
	})
	if err != nil {
		return nil, err
	}

	return cmd, nil
}
