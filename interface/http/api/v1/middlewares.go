package v1

import (
	"context"
	"github.com/go-http-utils/headers"
	"net/http"
	"strings"
)

const (
	BearerAuthorizationType = "Bearer"
)

func BearerAuthorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		authorization := request.Header.Get(headers.Authorization)

		if strings.TrimSpace(authorization) == "" {
			http.Error(writer, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		parts := strings.Split(authorization, BearerAuthorizationType)
		if len(parts) != 2 {
			http.Error(writer, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		next.ServeHTTP(writer, request.WithContext(context.WithValue(request.Context(), AccessTokenFieldName, strings.TrimSpace(parts[1]))))
	})
}
