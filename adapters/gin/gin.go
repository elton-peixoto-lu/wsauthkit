// Package ginwsauth adapts wsauthkit.Auth into a Gin middleware.
package ginwsauth

import (
	"github.com/gin-gonic/gin"

	"github.com/elton-peixoto-lu/wsauthkit"
)

// Middleware authenticates the handshake request and injects claims into
// the request context before calling the next Gin handler. On failure it
// writes the response via auth's configured ErrorHandler (or the default
// 401 handler) and aborts the chain.
func Middleware(auth *wsauthkit.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := auth.Authenticate(c.Request)
		if err != nil {
			auth.HandleError(c.Writer, c.Request, err)
			c.Abort()
			return
		}

		c.Request = c.Request.WithContext(wsauthkit.WithClaims(c.Request.Context(), claims))
		c.Next()
	}
}
