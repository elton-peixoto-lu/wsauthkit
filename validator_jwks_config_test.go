package wsauthkit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewJWTValidatorAppliesJWKSOverrideOptions(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	t.Cleanup(server.Close)

	var refreshErrCalls int

	validator, closer, err := NewJWTValidator(Config{
		Issuer:              "https://issuer.example.com",
		Audience:            "dashboard",
		JWKSURL:             server.URL,
		JWKSRequestTimeout:  2 * time.Second,
		JWKSRefreshInterval: time.Hour,
		JWKSRefreshErrorHandler: func(_ string, _ error) {
			refreshErrCalls++
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() {
		if err := closer(); err != nil {
			t.Fatalf("close: %v", err)
		}
	})

	if validator == nil {
		t.Fatal("expected non-nil validator")
	}
}
