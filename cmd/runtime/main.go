package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/wood-bison/fluent-task-runtime/internal/engine"
	"github.com/wood-bison/fluent-task-runtime/internal/httpapi"
	"github.com/wood-bison/fluent-task-runtime/internal/telemetry"
)

func main() {
	shutdownTelemetry, err := telemetry.Setup(context.Background())
	if err != nil {
		log.Fatalf("configure OpenTelemetry: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			log.Printf("flush OpenTelemetry: %v", err)
		}
	}()
	port := os.Getenv("RUNTIME_PORT")
	if port == "" {
		port = "48227"
	}
	address := ":" + port
	catalogue, err := engine.NewCatalogue()
	if err != nil {
		log.Fatalf("load task catalogue: %v", err)
	}
	server := &http.Server{Addr: address, Handler: httpapi.NewServer(catalogue)}
	log.Printf("fluent-task-runtime listening on %s", address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
