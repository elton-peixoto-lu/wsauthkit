//go:build e2e
// +build e2e

package wsauthkit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestE2EWebSocketHandshakeWithAuthorizationHeader(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "dashboard",
		SigningKey: []byte("secret"),
	})

	claims := defaultTestClaims()
	claims["role"] = "admin"
	token := signTestToken(t, []byte("secret"), claims)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer conn.Close()

		claims := MustClaims(r.Context())
		if err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s:%s", claims.Subject, claims.MustValue("role")))); err != nil {
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
	if string(message) != "user-123:admin" {
		t.Fatalf("unexpected websocket message: %q", message)
	}
}

func TestE2EWebSocketHandshakeWithSecWebSocketProtocol(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "dashboard",
		SigningKey: []byte("secret"),
	})

	claims := defaultTestClaims()
	token := signTestToken(t, []byte("secret"), claims)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer conn.Close()

		if err := conn.WriteMessage(websocket.TextMessage, []byte(MustClaims(r.Context()).Subject)); err != nil {
			t.Errorf("write websocket message: %v", err)
		}
	})))
	defer server.Close()

	websocketURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := *websocket.DefaultDialer
	dialer.Subprotocols = []string{"graphql-ws", "bearer", token}

	conn, _, err := dialer.Dial(websocketURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket message: %v", err)
	}
	if string(message) != "user-123" {
		t.Fatalf("unexpected websocket message: %q", message)
	}
}
