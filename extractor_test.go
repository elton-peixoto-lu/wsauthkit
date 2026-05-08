package wsauthkit

import (
	"net/http/httptest"
	"testing"
)

func TestAuthorizationHeaderExtractor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		headerValue   string
		expectedToken string
		expectedError error
	}{
		{
			name:          "bearer token",
			headerValue:   "Bearer header.token.value",
			expectedToken: "header.token.value",
		},
		{
			name:          "raw token",
			headerValue:   "header.token.value",
			expectedToken: "header.token.value",
		},
		{
			name:          "missing token",
			expectedError: ErrTokenMissing,
		},
		{
			name:          "invalid scheme format",
			headerValue:   "Bearer too many parts here",
			expectedError: ErrInvalidAuthorization,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest("GET", "/ws", nil)
			if testCase.headerValue != "" {
				request.Header.Set("Authorization", testCase.headerValue)
			}

			token, err := AuthorizationHeaderExtractor().ExtractToken(request)
			if err != testCase.expectedError {
				t.Fatalf("expected error %v, got %v", testCase.expectedError, err)
			}
			if token != testCase.expectedToken {
				t.Fatalf("expected token %q, got %q", testCase.expectedToken, token)
			}
		})
	}
}

func TestSecWebSocketProtocolExtractor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		headerValue   string
		expectedToken string
		expectedError error
	}{
		{
			name:          "bearer pair",
			headerValue:   "graphql-ws, bearer, a.b.c",
			expectedToken: "a.b.c",
		},
		{
			name:          "inline bearer token",
			headerValue:   "graphql-ws, bearer.a.b.c",
			expectedToken: "a.b.c",
		},
		{
			name:          "jwt token first",
			headerValue:   "a.b.c, graphql-ws",
			expectedToken: "a.b.c",
		},
		{
			name:          "missing token",
			headerValue:   "graphql-ws, json",
			expectedError: ErrTokenMissing,
		},
		{
			name:          "empty values only",
			headerValue:   " ,  , ",
			expectedError: ErrInvalidSecWebSocketHeader,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest("GET", "/ws", nil)
			request.Header.Set("Sec-WebSocket-Protocol", testCase.headerValue)

			token, err := SecWebSocketProtocolExtractor().ExtractToken(request)
			if err != testCase.expectedError {
				t.Fatalf("expected error %v, got %v", testCase.expectedError, err)
			}
			if token != testCase.expectedToken {
				t.Fatalf("expected token %q, got %q", testCase.expectedToken, token)
			}
		})
	}
}

func TestChainExtractorsStopsOnMalformedToken(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest("GET", "/ws", nil)
	request.Header.Set("Authorization", "Bearer too many parts here")
	request.Header.Set("Sec-WebSocket-Protocol", "graphql-ws, bearer, a.b.c")

	_, err := ChainExtractors(
		AuthorizationHeaderExtractor(),
		SecWebSocketProtocolExtractor(),
	).ExtractToken(request)
	if err != ErrInvalidAuthorization {
		t.Fatalf("expected ErrInvalidAuthorization, got %v", err)
	}
}

func TestDefaultExtractorReturnsMissingToken(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest("GET", "/ws", nil)

	_, err := DefaultExtractor().ExtractToken(request)
	if err != ErrTokenMissing {
		t.Fatalf("expected ErrTokenMissing, got %v", err)
	}
}
