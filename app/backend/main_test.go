package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestHandleLivez(t *testing.T) {
	app := &App{}
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	rr := httptest.NewRecorder()

	app.handleLivez(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestHandleReadyz_Uninitialized(t *testing.T) {
	app := &App{DB: nil}
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	app.handleReadyz(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 for uninitialized DB, got %d", rr.Code)
	}
}

func TestCorrelationMiddleware(t *testing.T) {
	t.Run("Generates Request ID when missing", func(t *testing.T) {
		var capturedReqID string
		dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedReqID, _ = r.Context().Value(requestIDKey).(string)
			w.WriteHeader(http.StatusOK)
		})

		handler := correlationMiddleware(dummyHandler)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		headerID := rr.Header().Get("X-Request-ID")
		if headerID == "" {
			t.Errorf("expected X-Request-ID header to be populated")
		}
		if capturedReqID == "" || capturedReqID != headerID {
			t.Errorf("expected context request_id (%s) to match header (%s)", capturedReqID, headerID)
		}
	})

	t.Run("Preserves incoming Request ID", func(t *testing.T) {
		incomingID := "custom-trace-id-12345"
		var capturedReqID string
		dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedReqID, _ = r.Context().Value(requestIDKey).(string)
			w.WriteHeader(http.StatusOK)
		})

		handler := correlationMiddleware(dummyHandler)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", incomingID)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Request-ID") != incomingID {
			t.Errorf("expected header X-Request-ID %s, got %s", incomingID, rr.Header().Get("X-Request-ID"))
		}
		if capturedReqID != incomingID {
			t.Errorf("expected context request_id %s, got %s", incomingID, capturedReqID)
		}
	})
}

func TestTaskValidation(t *testing.T) {
	app := &App{DB: nil}

	t.Run("Rejects empty title on POST", func(t *testing.T) {
		payload := `{"title": "", "description": "no title"}`
		req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(payload))
		rr := httptest.NewRecorder()

		app.handleTasks(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for empty title, got %d", rr.Code)
		}
	})

	t.Run("Rejects invalid JSON payload", func(t *testing.T) {
		payload := `{invalid-json`
		req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(payload))
		rr := httptest.NewRecorder()

		app.handleTasks(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for malformed json, got %d", rr.Code)
		}
	})
}
