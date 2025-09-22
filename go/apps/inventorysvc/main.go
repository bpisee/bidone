package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lee/BidOne/apps/inventorysvc/services"
	"github.com/lee/BidOne/apps/inventorysvc/store"
)

func main() {
	// Create the data store
	store := store.NewMemoryStore()

	// Create the service layer
	inventoryService := services.NewInventoryService(store)

	// Create the HTTP server with the service
	server := NewServer(inventoryService)

	// Configure server address
	address := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		address = ":" + port
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting inventory service on %s", address)
		log.Printf("Health check available at: http://localhost%s/health", address)
		log.Printf("API endpoints available at: http://localhost%s/api/v1/products", address)

		if err := server.Start(address); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server gracefully stopped")
	}
}
