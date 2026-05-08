package wsauthkit

import "context"

type contextKey string

const claimsContextKey contextKey = "wsauthkit.claims"

// WithClaims stores claims in a request context.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// ClaimsFromContext returns claims if the middleware already authenticated the request.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	return claims, ok
}

// MustClaims returns request claims or panics if they are missing.
func MustClaims(ctx context.Context) *Claims {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims == nil {
		panic("wsauthkit: claims missing from context")
	}

	return claims
}
