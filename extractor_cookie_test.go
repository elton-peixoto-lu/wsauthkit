package wsauthkit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCookieExtractor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		cookieValue   string
		setCookie     bool
		expectedToken string
		expectedError error
	}{
		{
			name:          "present cookie",
			setCookie:     true,
			cookieValue:   "header.token.value",
			expectedToken: "header.token.value",
		},
		{
			name:          "missing cookie",
			setCookie:     false,
			expectedError: ErrTokenMissing,
		},
		{
			name:          "empty cookie",
			setCookie:     true,
			cookieValue:   "",
			expectedError: ErrTokenMissing,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest("GET", "/ws", nil)
			if testCase.setCookie {
				request.AddCookie(&http.Cookie{Name: "session", Value: testCase.cookieValue})
			}

			token, err := CookieExtractor("session").ExtractToken(request)
			if err != testCase.expectedError {
				t.Fatalf("expected error %v, got %v", testCase.expectedError, err)
			}
			if token != testCase.expectedToken {
				t.Fatalf("expected token %q, got %q", testCase.expectedToken, token)
			}
		})
	}
}
