package container

import (
	"github.com/Diez37/go-skeleton/infrastructure/config"
	"github.com/Diez37/go-skeleton/infrastructure/repository"
	"github.com/diez37/go-packages/container"
	"github.com/go-playground/validator/v10"
)

func AddProvide(container container.Container) error {
	return container.Provides(
		repository.NewSql,
		config.NewToken,
		validator.New,
	)
}
