package server

import "net/http"

// HealthHandler exposes healthHandler to the external test package.
func HealthHandler(commit string) http.HandlerFunc {
	return healthHandler(commit)
}
