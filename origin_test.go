package wsauthkit

import (
	"net/http/httptest"
	"testing"
)

func TestAllowedOrigins(t *testing.T) {
	t.Parallel()

	validator := AllowedOrigins("https://app.example.com", "https://admin.example.com/")

	testCases := []struct {
		name        string
		origin      string
		expectError bool
	}{
		{name: "no origin header", origin: "", expectError: false},
		{name: "allowed origin", origin: "https://app.example.com", expectError: false},
		{name: "allowed origin case-insensitive", origin: "HTTPS://APP.EXAMPLE.COM", expectError: false},
		{name: "allowed origin with trailing slash config", origin: "https://admin.example.com", expectError: false},
		{name: "disallowed origin", origin: "https://evil.example.com", expectError: true},
		{name: "disallowed scheme", origin: "http://app.example.com", expectError: true},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest("GET", "/ws", nil)
			if testCase.origin != "" {
				request.Header.Set("Origin", testCase.origin)
			}

			err := validator.ValidateOrigin(request)
			if testCase.expectError && err != ErrOriginNotAllowed {
				t.Fatalf("expected ErrOriginNotAllowed, got %v", err)
			}
			if !testCase.expectError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
