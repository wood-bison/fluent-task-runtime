package main

import (
	"log"
	"net/http"
	"os"

	"github.com/wood-bison/fluent-task-runtime/internal/engine"
	"github.com/wood-bison/fluent-task-runtime/internal/httpapi"
)

func main() {
	port := os.Getenv("RUNTIME_PORT")
	if port == "" {
		port = "48227"
	}
	address := ":" + port
	server := &http.Server{Addr: address, Handler: httpapi.NewServer(engine.NewCatalogue())}
	log.Printf("fluent-task-runtime listening on %s", address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
