package wsauthkit

import (
	"net/http"
	"net/url"
	"strings"
)

// OriginValidator decides whether a handshake request's Origin is trusted.
//
// WebSocket handshakes are plain HTTP requests carried over the browser's
// automatic cookie/credential attachment, so without an explicit Origin
// check a malicious site can open an authenticated WebSocket against a
// user's session (cross-site WebSocket hijacking, CSWSH). This is
// especially relevant when using CookieExtractor.
type OriginValidator interface {
	ValidateOrigin(r *http.Request) error
}

// OriginValidatorFunc adapts a function into an OriginValidator.
type OriginValidatorFunc func(r *http.Request) error

func (f OriginValidatorFunc) ValidateOrigin(r *http.Request) error {
	return f(r)
}

// AllowedOrigins builds an OriginValidator from an explicit allowlist of
// scheme+host origins (e.g. "https://app.example.com"). Matching is exact
// and case-insensitive on scheme/host; comparisons ignore a trailing slash.
//
// A request with no Origin header (non-browser clients, same-process
// integration tests, native WebSocket clients) is allowed through, since the
// Origin header is only meaningful for browser-driven handshakes.
func AllowedOrigins(origins ...string) OriginValidator {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[normalizeOrigin(origin)] = struct{}{}
	}

	return OriginValidatorFunc(func(r *http.Request) error {
		raw := strings.TrimSpace(r.Header.Get("Origin"))
		if raw == "" {
			return nil
		}

		if _, ok := allowed[normalizeOrigin(raw)]; !ok {
			return ErrOriginNotAllowed
		}

		return nil
	})
}

func normalizeOrigin(origin string) string {
	trimmed := strings.TrimSuffix(strings.TrimSpace(origin), "/")

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.ToLower(trimmed)
	}

	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host)
}
