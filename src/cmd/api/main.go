package main

import (
	"kvs/src/internal/api/handler"
	"kvs/src/internal/kvs"
	"log"
	"net/http"
)

func main() {
	kvs := kvs.NewKeyValueStore()

	routes := handler.NewKeyValueHandler(kvs).Routes()
	log.Println("Hello from go-cloud!")
	log.Fatal(http.ListenAndServe(":8080", routes))
}
