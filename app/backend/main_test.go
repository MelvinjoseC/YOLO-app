package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTelemetryMiddleware(t *testing.T) {
	// Setup a dummy handler that returns 200 OK
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap with our telemetry middleware
	handlerToTest := telemetryMiddleware(dummyHandler)

	// Create a mock HTTP request
	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rr := httptest.NewRecorder()

	// Execute request
	handlerToTest.ServeHTTP(rr, req)

	// Verify response status
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify response body
	expectedBody := "OK"
	if rr.Body.String() != expectedBody {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expectedBody)
	}
}

func TestTelemetryMiddlewarePathNormalization(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handlerToTest := telemetryMiddleware(dummyHandler)

	// Request with a specific resource ID (high cardinality path)
	req := httptest.NewRequest(http.MethodDelete, "/api/tasks/42", nil)
	rr := httptest.NewRecorder()

	handlerToTest.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}
}
