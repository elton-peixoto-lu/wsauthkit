package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/elton-peixoto-lu/wsauthkit"
)

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
	fmt.Fprintf(w, "user=%s", claims.Subject)
}
