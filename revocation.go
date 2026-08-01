package wsauthkit

import (
	"context"
	"time"
)

// Revoker decides whether previously-valid claims must be treated as
// revoked (logout, token compromise, account ban, permission change).
//
// It is checked once during Authenticate/Middleware, at handshake time.
// For connections that stay open longer than the token's lifetime or
// longer than a revocation should take effect, pair it with Reverify to
// re-run the check periodically against the live connection.
type Revoker interface {
	IsRevoked(ctx context.Context, claims *Claims) (bool, error)
}

// RevokerFunc adapts a function into a Revoker.
type RevokerFunc func(ctx context.Context, claims *Claims) (bool, error)

func (f RevokerFunc) IsRevoked(ctx context.Context, claims *Claims) (bool, error) {
	return f(ctx, claims)
}

// Reverify periodically re-checks previously authenticated claims against
// token expiry and the configured Revoker, for connections that outlive a
// single handshake (long-lived WebSocket connections).
//
// It calls onInvalid and stops as soon as the claims are no longer valid
// (expired, revoked, or the Revoker errors), or when ctx is done. Run it in
// its own goroutine tied to the connection's lifetime:
//
//	connCtx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	go auth.Reverify(connCtx, claims, 30*time.Second, func(err error) {
//		cancel()
//		conn.Close()
//	})
func (a *Auth) Reverify(ctx context.Context, claims *Claims, interval time.Duration, onInvalid func(error)) {
	if a == nil || claims == nil || onInvalid == nil {
		return
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.checkStillValid(ctx, claims); err != nil {
				onInvalid(err)
				return
			}
		}
	}
}

func (a *Auth) checkStillValid(ctx context.Context, claims *Claims) error {
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return ErrInvalidToken
	}

	if a.revoker == nil {
		return nil
	}

	revoked, err := a.revoker.IsRevoked(ctx, claims)
	if err != nil {
		return err
	}
	if revoked {
		return ErrTokenRevoked
	}

	return nil
}
