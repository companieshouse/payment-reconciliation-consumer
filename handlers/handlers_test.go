package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/pat"
)

func TestUnitInit(t *testing.T) {
	r := pat.New()
	Init(r)

	req := httptest.NewRequest("GET", "/payment-reconciliation-consumer/healthcheck", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}
}
