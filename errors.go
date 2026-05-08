package wsauthkit

import "errors"

var (
	ErrAuthNotConfigured         = errors.New("wsauthkit: auth is not configured")
	ErrExtractorNotConfigured    = errors.New("wsauthkit: extractor is not configured")
	ErrValidatorNotConfigured    = errors.New("wsauthkit: validator is not configured")
	ErrMissingKeySource          = errors.New("wsauthkit: configure one key source via KeyFunc, SigningKey or JWKSURL")
	ErrTokenMissing              = errors.New("wsauthkit: token not found")
	ErrInvalidAuthorization      = errors.New("wsauthkit: invalid authorization header")
	ErrInvalidSecWebSocketHeader = errors.New("wsauthkit: invalid Sec-WebSocket-Protocol header")
	ErrInvalidToken              = errors.New("wsauthkit: invalid token")
)
