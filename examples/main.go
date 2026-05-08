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
}

func main() {
	auth, err := wsauthkit.NewAuth(wsauthkit.Config{
		Issuer:   "https://auth.company.com",
		Audience: "erp-backend",
		JWKSURL:  "https://auth.company.com/certs",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer auth.Close()

	http.Handle("/ws", auth.Middleware(http.HandlerFunc(wsHandler)))

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	claims := wsauthkit.MustClaims(r.Context())

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	defer conn.Close()

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)
			return
		}

		response := append([]byte("user="+claims.Subject+" "), payload...)
		if err := conn.WriteMessage(messageType, response); err != nil {
			log.Printf("write error: %v", err)
			return
		}
	}
}
