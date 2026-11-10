package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthReportsCommit(t *testing.T) {
	rec := httptest.NewRecorder()
	healthHandler("abc123")(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body["status"] != "ok" || body["commit"] != "abc123" {
		t.Errorf("body = %v, want status ok and commit abc123", body)
	}
}
