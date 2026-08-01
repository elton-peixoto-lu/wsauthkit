// Package echowsauth adapts wsauthkit.Auth into an Echo middleware.
package echowsauth

import (
	"github.com/labstack/echo/v4"

	"github.com/elton-peixoto-lu/wsauthkit"
)

// Middleware authenticates the handshake request and injects claims into
// the request context before calling the next Echo handler. On failure it
// writes the response via auth's configured ErrorHandler (or the default
// 401 handler) and stops the chain.
func Middleware(auth *wsauthkit.Auth) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			request := c.Request()

			claims, err := auth.Authenticate(request)
			if err != nil {
				auth.HandleError(c.Response().Writer, request, err)
				return nil
			}

			c.SetRequest(request.WithContext(wsauthkit.WithClaims(request.Context(), claims)))
			return next(c)
		}
	}
}
