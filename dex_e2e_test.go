//go:build dex_e2e
// +build dex_e2e

package wsauthkit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

const (
	dexIssuer       = "http://127.0.0.1:5556/dex"
	dexJWKSURL      = "http://127.0.0.1:5556/dex/keys"
	dexTokenURL     = "http://127.0.0.1:5556/dex/token"
	dexClientID     = "wsauthkit-e2e"
	dexClientSecret = "wsauthkit-secret"
	dexUsername     = "e2e-user@example.com"
	dexPassword     = "secret123"
)

func TestDexE2EWebSocketHandshakeWithRealDexToken(t *testing.T) {
	t.Parallel()

	token := fetchDexIDToken(t)
	auth := newTestAuth(t, Config{
		Issuer:   dexIssuer,
		Audience: dexClientID,
		JWKSURL:  dexJWKSURL,
	})

	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	server := httptest.NewServer(auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer conn.Close()

		claims := MustClaims(r.Context())
		email, _ := claims.Value("email")
		payload := claims.Subject + "|" + emailString(email)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
			t.Errorf("write websocket message: %v", err)
		}
	})))
	defer server.Close()

	websocketURL := "ws" + strings.TrimPrefix(server.URL, "http")
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)

	conn, _, err := websocket.DefaultDialer.Dial(websocketURL, header)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket message: %v", err)
	}
	parts := strings.SplitN(string(message), "|", 2)
	if len(parts) != 2 {
		t.Fatalf("unexpected payload format: %q", message)
	}
	if strings.TrimSpace(parts[0]) == "" {
		t.Fatalf("expected non-empty subject, got %q", parts[0])
	}
	if parts[1] != dexUsername {
		t.Fatalf("unexpected email claim: %q", parts[1])
	}
}

func TestDexE2ERejectsTamperedToken(t *testing.T) {
	t.Parallel()

	token := fetchDexIDToken(t) + "tampered"
	auth := newTestAuth(t, Config{
		Issuer:   dexIssuer,
		Audience: dexClientID,
		JWKSURL:  dexJWKSURL,
	})

	handler := auth.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not be reached")
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	if body := recorder.Body.String(); body != "invalid authentication token\n" {
		t.Fatalf("unexpected body %q", body)
	}
}

func fetchDexIDToken(t *testing.T) string {
	t.Helper()

	client := &http.Client{Timeout: 10 * time.Second}
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", dexUsername)
	form.Set("password", dexPassword)
	form.Set("scope", "openid profile email")

	request, err := http.NewRequest(http.MethodPost, dexTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("new token request: %v", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(dexClientID, dexClientSecret)

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("request dex token: %v", err)
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read token response: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("dex token request failed: status=%d body=%s", response.StatusCode, payload)
	}

	var tokenResponse struct {
		IDToken string `json:"id_token"`
	}
	if err := json.Unmarshal(payload, &tokenResponse); err != nil {
		t.Fatalf("parse token response: %v", err)
	}
	if tokenResponse.IDToken == "" {
		t.Fatalf("empty id token in response: %s", payload)
	}

	return tokenResponse.IDToken
}

func emailString(value any) string {
	text, _ := value.(string)
	return text
}
