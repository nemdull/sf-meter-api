package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/nemdull/sf-meter-api/registry"
)

func main() {
	// Initialize registry and dependencies
	reg, err := registry.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize registry: %v\n", err)
		os.Exit(1)
	}

	// Create handler
	handler, err := reg.NewTechHandler()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize handler: %v\n", err)
		os.Exit(1)
	}

	// Setup routes using stdlib net/http with Go 1.22+ enhanced ServeMux
	mux := http.NewServeMux()
	mux.HandleFunc("GET /techs", handler.GetTechs)
	mux.HandleFunc("GET /techs/{id}", handler.GetTechByID)
	mux.HandleFunc("GET /health", handleHealth)

	// Get port from environment, default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Validate port is a number
	if _, err := strconv.Atoi(port); err != nil {
		fmt.Fprintf(os.Stderr, "invalid PORT: %v\n", err)
		os.Exit(1)
	}

	addr := net.JoinHostPort("0.0.0.0", port)
	fmt.Printf("Starting sf-meter API on http://localhost:%s\n", port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

// handleHealth responds with 200 OK for health checks
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ok"}`)
}
