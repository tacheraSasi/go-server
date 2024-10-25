package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerResponse(t *testing.T) {
	server := NewServer("8080")
	server.AddRoute("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Test successful"))
	})

	req := httptest.NewRequest("GET", "http://localhost:8080/", nil)
	// ResponseRecorder captures the response for verification.
	rec := httptest.NewRecorder()

	// Serve HTTP request using Server's mux.
	server.mux.ServeHTTP(rec, req)

	// Checking if response status code is 200 OK.
	if status := rec.Result().StatusCode; status != http.StatusOK {
		t.Errorf("Expected status code 200, got %v", status)
	}

	// Check response body for expected content.
	expected := "Test successful"
	if rec.Body.String() != expected {
		t.Errorf("Expected response body '%s', got '%s'", expected, rec.Body.String())
	}
}
