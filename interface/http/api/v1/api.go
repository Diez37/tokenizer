package v1

import (
	"encoding/json"
	"fmt"
	"github.com/Diez37/go-skeleton/application"
	"github.com/Diez37/go-skeleton/domain"
	"github.com/Diez37/go-skeleton/infrastructure/config"
	"github.com/diez37/go-packages/log"
	"github.com/go-http-utils/headers"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/ldez/mimetype"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"io"
	"net"
	"net/http"
	"time"
)

type API interface {
	Create(writer http.ResponseWriter, request *http.Request) // Create
	Read(writer http.ResponseWriter, request *http.Request)   // Parse
	Update(writer http.ResponseWriter, request *http.Request) // RefreshToken
	Delete(writer http.ResponseWriter, request *http.Request) // Disable

	Validation(writer http.ResponseWriter, request *http.Request)
	DeleteAll(writer http.ResponseWriter, request *http.Request)
}

type api struct {
	config *config.Token

	logger    log.Logger
	service   application.Token
	validator *validator.Validate
	tracer    trace.Tracer
}

func NewApi(
	config *config.Token,
	logger log.Logger,
	service application.Token,
	validator *validator.Validate,
	tracer trace.Tracer,
) API {
	return &api{config: config, logger: logger, service: service, validator: validator, tracer: tracer}
}

func (api *api) Create(writer http.ResponseWriter, request *http.Request) {
	ctx, span := api.tracer.Start(request.Context(), "api.token.create")
	defer span.End()

	span.SetAttributes(attribute.Int("version", 1))

	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		api.logger.Error(err)
		return
	}

	model := Create{}
	if err := json.Unmarshal(body, &model); err != nil {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		api.logger.Error(err)
		return
	}

	if err := api.validator.Struct(model); err != nil {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		api.logger.Error(err)
		return
	}

	refreshToken, accessToken, err := api.service.Create(ctx, &domain.RefreshToken{
		Login:       model.Login,
		Ip:          ctx.Value(IpFieldName).(net.IP),
		Fingerprint: model.Fingerprint,
		UserAgent:   ctx.Value(UserAgentFieldName).(string),
	})
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		api.logger.Error(err)
		return
	}

	http.SetCookie(writer, &http.Cookie{
		Name:     RefreshTokenFieldName,
		Value:    refreshToken.UUID.String(),
		Path:     request.RequestURI,
		Expires:  time.Now().In(time.UTC).Add(api.config.RefreshLifetime),
		HttpOnly: true,
	})

	writer.Header().Add(headers.Authorization, fmt.Sprintf("%s %s", BearerAuthorizationType, accessToken))
	writer.WriteHeader(http.StatusOK)
}

func (api *api) Read(writer http.ResponseWriter, request *http.Request) {
	ctx, span := api.tracer.Start(request.Context(), "api.token.read")
	defer span.End()

	span.SetAttributes(attribute.Int("version", 1))

	token, err := api.service.Parse(ctx, ctx.Value(AccessTokenFieldName).(string))
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		api.logger.Error(err)
		return
	}

	body, err := json.Marshal(&AccessToken{
		Login:     token.Login,
		ExpiresIn: token.ExpiresIn,
	})
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		api.logger.Error(err)
		return
	}

	if _, err := writer.Write(body); err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		api.logger.Error(err)
		return
	}

	writer.Header().Add(headers.ContentType, mimetype.ApplicationJSON)
	writer.WriteHeader(http.StatusOK)
}

func (api *api) Update(writer http.ResponseWriter, request *http.Request) {
	ctx, span := api.tracer.Start(request.Context(), "api.token.update")
	defer span.End()

	span.SetAttributes(attribute.Int("version", 1))

	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		api.logger.Error(err)
		return
	}

	model := RefreshToken{}

	if err := json.Unmarshal(body, &model); err != nil {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		api.logger.Error(err)
		return
	}

	if err := api.validator.Struct(model); err != nil {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		api.logger.Error(err)
		return
	}

	refreshToken, accessToken, err := api.service.Refresh(ctx, &domain.RefreshToken{
		UUID:        ctx.Value(RefreshTokenFieldName).(uuid.UUID),
		Ip:          ctx.Value(IpFieldName).(net.IP),
		Fingerprint: model.Fingerprint,
		UserAgent:   ctx.Value(UserAgentFieldName).(string),
	})
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		api.logger.Error(err)
		return
	}

	http.SetCookie(writer, &http.Cookie{
		Name:     RefreshTokenFieldName,
		Value:    refreshToken.UUID.String(),
		Path:     request.RequestURI,
		Expires:  time.Now().In(time.UTC).Add(api.config.RefreshLifetime),
		HttpOnly: true,
	})

	writer.Header().Add(headers.Authorization, fmt.Sprintf("%s %s", BearerAuthorizationType, accessToken))
	writer.WriteHeader(http.StatusOK)
}

func (api *api) Delete(writer http.ResponseWriter, request *http.Request) {
	ctx, span := api.tracer.Start(request.Context(), "api.token.delete")
	defer span.End()

	span.SetAttributes(attribute.Int("version", 1))

	if err := api.service.Disable(ctx, ctx.Value(RefreshTokenFieldName).(uuid.UUID)); err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		api.logger.Error(err)
		return
	}

	http.SetCookie(writer, &http.Cookie{
		Name:     RefreshTokenFieldName,
		Path:     request.RequestURI,
		Expires:  time.Now().In(time.UTC),
		HttpOnly: true,
	})

	writer.WriteHeader(http.StatusAccepted)
}

func (api *api) Validation(writer http.ResponseWriter, request *http.Request) {
	ctx, span := api.tracer.Start(request.Context(), "api.token.validation")
	defer span.End()

	span.SetAttributes(attribute.Int("version", 1))

	if err := api.service.Validation(ctx, ctx.Value(AccessTokenFieldName).(string)); err != nil {
		http.Error(writer, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		api.logger.Error(err)
		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (api *api) DeleteAll(writer http.ResponseWriter, request *http.Request) {
	ctx, span := api.tracer.Start(request.Context(), "api.token.delete.all")
	defer span.End()

	span.SetAttributes(attribute.Int("version", 1))

	accessToken, err := api.service.Parse(ctx, ctx.Value(AccessTokenFieldName).(string))
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		api.logger.Error(err)
		return
	}

	if err := api.service.DisableAll(ctx, accessToken.Login, ctx.Value(RefreshTokenFieldName).(uuid.UUID)); err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		api.logger.Error(err)
		return
	}

	writer.WriteHeader(http.StatusAccepted)
}
