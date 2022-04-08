package application

import (
	"context"
	"errors"
	"fmt"
	"github.com/Diez37/go-skeleton/domain"
	"github.com/Diez37/go-skeleton/infrastructure/config"
	"github.com/Diez37/go-skeleton/infrastructure/repository"
	"github.com/diez37/go-packages/clients/db"
	"github.com/diez37/go-packages/log"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"github.com/thoas/go-funk"
	"go.opentelemetry.io/otel/trace"
	"time"
)

const (
	ExpiresInJwtFieldName = "exp"
	LoginJwtFieldName     = "login"
)

var (
	AccessDeniedError = errors.New("access denied")
)

type Token interface {
	Create(ctx context.Context, token *domain.RefreshToken) (*domain.RefreshToken, string, error)
	Refresh(ctx context.Context, token *domain.RefreshToken) (*domain.RefreshToken, string, error)
	DisableAll(ctx context.Context, login uuid.UUID, exclude ...uuid.UUID) error
	Disable(ctx context.Context, uuid uuid.UUID) error
	Validation(ctx context.Context, token string) error
	Parse(ctx context.Context, token string) (*domain.JwtClaims, error)
}

type token struct {
	logger log.Logger
	config *config.Token
	secret []byte

	finder  repository.Finder
	saver   repository.Saver
	blocker repository.Blocker
	parser  *jwt.Parser

	tracer trace.Tracer
}

func NewToken(
	config *config.Token,
	logger log.Logger,
	finder repository.Finder,
	saver repository.Saver,
	blocker repository.Blocker,
	tracer trace.Tracer,
) Token {
	return &token{
		config:  config,
		finder:  finder,
		saver:   saver,
		blocker: blocker,
		logger:  logger,
		secret:  []byte(config.Secret),
		parser:  new(jwt.Parser),
		tracer:  tracer,
	}
}

func (service *token) Create(ctx context.Context, token *domain.RefreshToken) (*domain.RefreshToken, string, error) {
	ctx, span := service.tracer.Start(ctx, "service.token.create")
	defer span.End()

	service.logger.Infof("token.service: create new token for login '%s'", token.Login.String())

	tokens, err := service.finder.FindByLogin(ctx, token.Login)
	if err != nil && err != db.RecordNotFoundError {
		return nil, "", err
	}

	if len(tokens) >= int(service.config.MaximumTokens) {
		if err := service.blocker.BlockByUUID(ctx, tokens[0].UUID); err != nil {
			return nil, "", err
		}
	}

	return service.generate(ctx, token)
}

func (service *token) Refresh(ctx context.Context, token *domain.RefreshToken) (*domain.RefreshToken, string, error) {
	ctx, span := service.tracer.Start(ctx, "service.token.refresh")
	defer span.End()

	refreshToken, err := service.finder.FindByUUID(ctx, token.UUID)
	if err != nil && err != db.RecordNotFoundError {
		return nil, "", err
	}

	if err == db.RecordNotFoundError {
		return nil, "", AccessDeniedError
	}

	if refreshToken.ExpiresIn.Sub(time.Now().In(time.UTC)) <= 0 {
		if err := service.blocker.BlockByUUID(ctx, refreshToken.UUID); err != nil {
			return nil, "", err
		}

		return nil, "", AccessDeniedError
	}

	for _, fieldForCheck := range service.config.RefreshCheckFields {
		switch fieldForCheck {
		case config.TokenRefreshFieldIp:
			if refreshToken.Ip != token.Ip.String() {
				err = AccessDeniedError
			}
		case config.TokenRefreshFieldFingerprint:
			if refreshToken.Fingerprint != token.Fingerprint {
				err = AccessDeniedError
			}
		case config.TokenRefreshFieldUserAgent:
			if refreshToken.UserAgent != token.UserAgent {
				err = AccessDeniedError
			}
		}
	}
	if err != nil {
		switch service.config.AccessViolation {
		case config.TokensAccessViolationActionDisableAll:
			if err := service.DisableAll(ctx, refreshToken.Login); err != nil {
				return nil, "", err
			}
		case config.TokensAccessViolationActionDisableCurrent:
			if err := service.blocker.BlockByUUID(ctx, refreshToken.UUID); err != nil {
				return nil, "", err
			}
		}

		return nil, "", err
	}

	if err := service.blocker.BlockByUUID(ctx, refreshToken.UUID); err != nil {
		return nil, "", err
	}

	token.Login = refreshToken.Login

	return service.generate(ctx, token)
}

func (service *token) generate(ctx context.Context, token *domain.RefreshToken) (*domain.RefreshToken, string, error) {
	ctx, span := service.tracer.Start(ctx, "service.token.generate")
	defer span.End()

	var err error

	token.UUID, err = uuid.NewRandom()
	if err != nil {
		return nil, "", err
	}

	now := time.Now().In(time.UTC)

	err = service.saver.Insert(ctx, &repository.RefreshToken{
		UUID:        token.UUID,
		Login:       token.Login,
		Ip:          token.Ip.String(),
		Fingerprint: token.Fingerprint,
		UserAgent:   token.UserAgent,
		CreatedAt:   now,
		ExpiresIn:   now.Add(service.config.RefreshLifetime),
	})
	if err != nil {
		return nil, "", err
	}

	jsonToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		LoginJwtFieldName:     token.Login.String(),
		ExpiresInJwtFieldName: now.Add(service.config.AccessLifetime).Unix(),
	})

	jwt, err := jsonToken.SignedString(service.secret)
	if err != nil {
		return nil, "", err
	}

	return token, jwt, nil
}

func (service *token) DisableAll(ctx context.Context, login uuid.UUID, exclude ...uuid.UUID) error {
	ctx, span := service.tracer.Start(ctx, "service.token.disable.all")
	defer span.End()

	tokens, err := service.finder.FindByLogin(ctx, login)
	if err != nil && err != db.RecordNotFoundError {
		return err
	}

	if err == db.RecordNotFoundError {
		return nil
	}

	for _, excludeTokenUUID := range exclude {
		index := funk.IndexOf(tokens, func(token *repository.RefreshToken) bool { return token.UUID == excludeTokenUUID })
		if index == -1 {
			continue
		}

		tokens = append(tokens[:index], tokens[index+1:]...)
	}

	uuids := make([]uuid.UUID, 0, len(tokens))
	for _, token := range tokens {
		uuids = append(uuids, token.UUID)
	}

	return service.blocker.BlockByUUID(ctx, uuids...)
}

func (service *token) Disable(ctx context.Context, uuid uuid.UUID) error {
	ctx, span := service.tracer.Start(ctx, "service.token.disable")
	defer span.End()

	return service.blocker.BlockByUUID(ctx, uuid)
}

func (service *token) Validation(ctx context.Context, token string) error {
	ctx, span := service.tracer.Start(ctx, "service.token.validation")
	defer span.End()

	jwtToken, err := service.parse(ctx, token)
	if err != nil {
		return err
	}

	if !jwtToken.Valid {
		return AccessDeniedError
	}

	return nil
}

func (service *token) Parse(ctx context.Context, token string) (*domain.JwtClaims, error) {
	ctx, span := service.tracer.Start(ctx, "service.token.parse")
	defer span.End()

	jwtToken, err := service.parse(ctx, token)
	if err != nil && jwtToken == nil {
		return nil, err
	}

	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New(fmt.Sprintf("jwtToken.Claims: not conversion to jwt.MapClaims, type '%T'", jwtToken.Claims))
	}

	jwtClaims := &domain.JwtClaims{}

	if login, exist := claims[LoginJwtFieldName]; !exist {
		return nil, errors.New(fmt.Sprintf("jwtToken.Claims: field '%s' not found", LoginJwtFieldName))
	} else {
		s, err := cast.ToStringE(login)
		if err != nil {
			return nil, err
		}

		jwtClaims.Login, err = uuid.Parse(s)
		if err != nil {
			return nil, err
		}
	}

	if nbf, exist := claims[ExpiresInJwtFieldName]; !exist {
		return nil, errors.New(fmt.Sprintf("jwtToken.Claims: field '%s' not found", ExpiresInJwtFieldName))
	} else {
		n, err := cast.ToInt64E(nbf)
		if err != nil {
			return nil, err
		}

		jwtClaims.ExpiresIn = time.Unix(n, 0).In(time.UTC)
	}

	return jwtClaims, nil
}

func (service *token) parse(ctx context.Context, token string) (*jwt.Token, error) {
	_, span := service.tracer.Start(ctx, "service.token.parse.jwt")
	defer span.End()

	return service.parser.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return service.secret, nil
	})
}
