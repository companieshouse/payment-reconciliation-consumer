package handlers

import "net/http"

// HealthCheck returns the health of the application.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}