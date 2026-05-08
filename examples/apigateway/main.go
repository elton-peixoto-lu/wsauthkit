package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/elton-peixoto-lu/wsauthkit"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	Subprotocols: []string{"graphql-ws"},
}

func main() {
	auth, err := wsauthkit.NewAuth(wsauthkit.Config{
		Issuer:   "https://auth.company.com",
		Audience: "apigateway-ws",
		JWKSURL:  "https://auth.company.com/certs",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer auth.Close()

	http.Handle("/ws", auth.Middleware(http.HandlerFunc(wsHandler)))

	log.Println("api-gateway example listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	claims := wsauthkit.MustClaims(r.Context())

	conn, err := upgrader.Upgrade(w, r, http.Header{
		"Sec-WebSocket-Protocol": []string{"graphql-ws"},
	})
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte("authenticated user="+claims.Subject)); err != nil {
		log.Printf("write error: %v", err)
		return
	}

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}

		if err := conn.WriteMessage(messageType, payload); err != nil {
			return
		}
	}
}
