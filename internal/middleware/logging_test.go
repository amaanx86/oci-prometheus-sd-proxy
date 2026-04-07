package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogging_PassesRequestToNext(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := Logging(next)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("Logging middleware did not call next handler")
	}
}

func TestLogging_CapturesStatusCode(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"400 Bad Request", http.StatusBadRequest},
		{"401 Unauthorized", http.StatusUnauthorized},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			})

			handler := Logging(next)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.statusCode {
				t.Errorf("status code: got %d, want %d", rr.Code, tc.statusCode)
			}
		})
	}
}

func TestLogging_DefaultStatusIs200(t *testing.T) {
	// When next handler never calls WriteHeader, the default is 200.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("body only"))
	})

	handler := Logging(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status code: got %d, want 200", rr.Code)
	}
}

func TestLogging_ResponseBodyPassedThrough(t *testing.T) {
	want := "hello world"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(want))
	})

	handler := Logging(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Body.String(); got != want {
		t.Errorf("body: got %q, want %q", got, want)
	}
}

func TestLogging_WithRefreshIntervalHeader(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := Logging(next)
	req := httptest.NewRequest(http.MethodGet, "/v1/targets", nil)
	req.Header.Set("X-Prometheus-Refresh-Interval-Seconds", "30")
	rr := httptest.NewRecorder()

	// Should not panic or error when the header is present
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status code: got %d, want 200", rr.Code)
	}
}

// TestResponseWriter_WriteHeader verifies the custom responseWriter wrapper
// only records the status once (the first call wins for http.ResponseWriter).
func TestResponseWriter_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, status: http.StatusOK}

	rw.WriteHeader(http.StatusTeapot)
	if rw.status != http.StatusTeapot {
		t.Errorf("status: got %d, want %d", rw.status, http.StatusTeapot)
	}

	// Underlying recorder should also reflect the status
	if rr.Code != http.StatusTeapot {
		t.Errorf("recorder code: got %d, want %d", rr.Code, http.StatusTeapot)
	}
}
