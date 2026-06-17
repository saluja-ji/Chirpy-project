package auth

import (
	"errors"
	"net/http"
	"strings"
)

func GetBearerToken(header http.Header) (string, error) {
	authHeader := header.Get("Authorization")

	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}

	const prefix = "Bearer "

	if !strings.HasPrefix(authHeader, prefix) {
		return "", errors.New("invalid authorization header")
	}

	return strings.TrimPrefix(authHeader, prefix), nil
}
