package websocket

import (
	"log"
	"net/http"
	"testing"
)

func TestConnection(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		NewWebsocketHandler(func(data []byte) (Opcode, []byte, error) {
			return Text, data, nil
		}).ServeHTTP(w, r)
	})

	log.Println("Starting asdf")
	http.ListenAndServe(":8080", mux)
}
