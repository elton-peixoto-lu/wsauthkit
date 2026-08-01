// Package fiberwsauth adapts wsauthkit.Auth into a Fiber middleware.
//
// Fiber runs on fasthttp rather than net/http, so this adapter bridges a
// fiber.Ctx's handshake request into a *http.Request for wsauthkit.Auth,
// and stores the resulting claims in fiber.Ctx locals under ClaimsLocalsKey.
package fiberwsauth

import (
	"net/http"
	"net/http/httptest"

	"github.com/gofiber/fiber/v2"

	"github.com/elton-peixoto-lu/wsauthkit"
)

// ClaimsLocalsKey is the fiber.Ctx locals key under which authenticated
// claims are stored by Middleware.
const ClaimsLocalsKey = "wsauthkit.claims"

// Middleware authenticates the handshake request and stores claims in
// fiber.Ctx locals (retrieve with ClaimsFromFiber) before calling the next
// handler. On failure it writes the response via auth's configured
// ErrorHandler (or the default 401 handler) and stops the chain.
func Middleware(auth *wsauthkit.Auth) fiber.Handler {
	return func(c *fiber.Ctx) error {
		request, err := toHTTPRequest(c)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "malformed request")
		}

		claims, err := auth.Authenticate(request)
		if err != nil {
			recorder := httptest.NewRecorder()
			auth.HandleError(recorder, request, err)

			c.Status(recorder.Code)
			for key, values := range recorder.Header() {
				for _, value := range values {
					c.Set(key, value)
				}
			}
			return c.SendString(recorder.Body.String())
		}

		c.Locals(ClaimsLocalsKey, claims)
		return c.Next()
	}
}

// ClaimsFromFiber returns the claims stored by Middleware, if present.
func ClaimsFromFiber(c *fiber.Ctx) (*wsauthkit.Claims, bool) {
	claims, ok := c.Locals(ClaimsLocalsKey).(*wsauthkit.Claims)
	return claims, ok
}

func toHTTPRequest(c *fiber.Ctx) (*http.Request, error) {
	request := httptest.NewRequest(c.Method(), c.OriginalURL(), nil)

	c.Context().Request.Header.VisitAll(func(key, value []byte) {
		request.Header.Add(string(key), string(value))
	})

	c.Context().Request.Header.VisitAllCookie(func(key, value []byte) {
		request.AddCookie(&http.Cookie{Name: string(key), Value: string(value)})
	})

	request = request.WithContext(c.Context())

	return request, nil
}
