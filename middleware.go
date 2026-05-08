package wsauthkit

import (
	"errors"
	"net/http"
)

// ErrorHandler writes the HTTP response for authentication failures.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// Middleware authenticates the request before invoking the wrapped handler.
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errorHandler := DefaultErrorHandler
		if a != nil && a.errorHandler != nil {
			errorHandler = a.errorHandler
		}

		claims, err := a.Authenticate(r)
		if err != nil {
			errorHandler(w, r, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
	})
}

// DefaultErrorHandler returns 401 responses without leaking token internals.
func DefaultErrorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	status := http.StatusUnauthorized
	message := "unauthorized"

	switch {
	case errors.Is(err, ErrInvalidAuthorization), errors.Is(err, ErrInvalidSecWebSocketHeader):
		message = "malformed authentication token"
	case errors.Is(err, ErrTokenMissing):
		message = "authentication token missing"
	case errors.Is(err, ErrInvalidToken):
		message = "invalid authentication token"
	}

	http.Error(w, message, status)
}
