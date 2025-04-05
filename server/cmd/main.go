package main

import (
	"log"
	"net/http"
	"os"

	"github.com/MattSilvaa/powhunter/internal/handlers"
)

func main() {
	h, err := handlers.NewHandlers()
	if err != nil {
		log.Fatalf("Failed to initialize handlers: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/resorts", h.Resort.GetAllResorts)
	mux.HandleFunc("api/createAlert", h.Alert.CreateAlert)

	handler := corsMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := "*"
		if os.Getenv("ENVIRONMENT") == "production" {
			allowedOrigin = "https://powhunter.onrender.com"
		}
		origin := r.Header.Get("Origin")
		if origin == allowedOrigin || allowedOrigin == "*" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
