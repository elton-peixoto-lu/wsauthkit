package wsauthkit

import (
	"net/http"
	"strings"
)

// TokenExtractor extracts a token from a WebSocket handshake request.
type TokenExtractor interface {
	ExtractToken(r *http.Request) (string, error)
}

// TokenExtractorFunc adapts a function into a TokenExtractor.
type TokenExtractorFunc func(r *http.Request) (string, error)

func (extractFunc TokenExtractorFunc) ExtractToken(request *http.Request) (string, error) {
	return extractFunc(request)
}

// DefaultExtractor tries Authorization first and then Sec-WebSocket-Protocol.
func DefaultExtractor() TokenExtractor {
	return ChainExtractors(
		AuthorizationHeaderExtractor(),
		SecWebSocketProtocolExtractor(),
	)
}

// ChainExtractors runs extractors in order and returns the first token found.
func ChainExtractors(extractors ...TokenExtractor) TokenExtractor {
	filtered := make([]TokenExtractor, 0, len(extractors))
	for _, extractor := range extractors {
		if extractor != nil {
			filtered = append(filtered, extractor)
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	return TokenExtractorFunc(func(r *http.Request) (string, error) {
		var lastError error

		for _, extractor := range filtered {
			token, err := extractor.ExtractToken(r)
			switch {
			case err == nil:
				return token, nil
			case err == ErrTokenMissing:
				lastError = err
				continue
			default:
				return "", err
			}
		}

		if lastError == nil {
			lastError = ErrTokenMissing
		}

		return "", lastError
	})
}

// AuthorizationHeaderExtractor extracts a bearer token from Authorization.
func AuthorizationHeaderExtractor() TokenExtractor {
	return TokenExtractorFunc(func(request *http.Request) (string, error) {
		headerValue := strings.TrimSpace(request.Header.Get("Authorization"))
		if headerValue == "" {
			return "", ErrTokenMissing
		}

		parts := strings.Fields(headerValue)
		switch {
		case len(parts) == 1:
			return parts[0], nil
		case len(parts) == 2 && strings.EqualFold(parts[0], "Bearer"):
			return parts[1], nil
		default:
			return "", ErrInvalidAuthorization
		}
	})
}

// SecWebSocketProtocolExtractor extracts a JWT from Sec-WebSocket-Protocol.
func SecWebSocketProtocolExtractor() TokenExtractor {
	return TokenExtractorFunc(func(request *http.Request) (string, error) {
		headerValue := strings.TrimSpace(request.Header.Get("Sec-WebSocket-Protocol"))
		if headerValue == "" {
			return "", ErrTokenMissing
		}

		protocolValues := splitHeaderList(headerValue)
		if len(protocolValues) == 0 {
			return "", ErrInvalidSecWebSocketHeader
		}

		for index, protocolValue := range protocolValues {
			lowerValue := strings.ToLower(protocolValue)

			switch {
			case looksLikeJWT(protocolValue):
				return protocolValue, nil
			case (lowerValue == "bearer" || lowerValue == "jwt" || lowerValue == "token") && index+1 < len(protocolValues):
				nextProtocolValue := protocolValues[index+1]
				if looksLikeJWT(nextProtocolValue) {
					return nextProtocolValue, nil
				}
			case strings.HasPrefix(lowerValue, "bearer."):
				token := strings.TrimSpace(protocolValue[len("bearer."):])
				if looksLikeJWT(token) {
					return token, nil
				}
			case strings.HasPrefix(lowerValue, "jwt."):
				token := strings.TrimSpace(protocolValue[len("jwt."):])
				if looksLikeJWT(token) {
					return token, nil
				}
			}
		}

		return "", ErrTokenMissing
	})
}

func splitHeaderList(value string) []string {
	chunks := strings.Split(value, ",")
	out := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk != "" {
			out = append(out, chunk)
		}
	}
	return out
}

func looksLikeJWT(value string) bool {
	parts := strings.Split(value, ".")
	return len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != ""
}
