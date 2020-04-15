package main

import (
	"log"
	"net/http"

	"ddns/server"
)

func main() {
	log.Fatal(http.ListenAndServe(":8080", server.Router()))
}
