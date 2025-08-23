package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnnynv/RepoSentry/pkg/logger"
)

func TestCORS(t *testing.T) {
	t.Run("TestCORSHeaders", func(t *testing.T) {
		handler := CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin: *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
		}

		if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST, PUT, DELETE, OPTIONS" {
			t.Errorf("Expected Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS, got %s", w.Header().Get("Access-Control-Allow-Methods"))
		}

		if w.Header().Get("Access-Control-Allow-Headers") != "Content-Type, Authorization" {
			t.Errorf("Expected Access-Control-Allow-Headers: Content-Type, Authorization, got %s", w.Header().Get("Access-Control-Allow-Headers"))
		}
	})

	t.Run("TestOPTIONSRequest", func(t *testing.T) {
		handler := CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Next handler should not be called for OPTIONS request")
		}))

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("TestNormalRequest", func(t *testing.T) {
		called := false
		handler := CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if !called {
			t.Error("Next handler should be called for normal request")
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestRequestLogger(t *testing.T) {
	t.Run("TestRequestLogging", func(t *testing.T) {
		log := logger.GetDefaultLogger().WithField("test", "middleware")
		handler := RequestLogger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set("User-Agent", "test-agent")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("TestRequestLoggingWithError", func(t *testing.T) {
		log := logger.GetDefaultLogger().WithField("test", "middleware")
		handler := RequestLogger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))

		req := httptest.NewRequest("POST", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

func TestRecovery(t *testing.T) {
	t.Run("TestRecoveryFromPanic", func(t *testing.T) {
		log := logger.GetDefaultLogger().WithField("test", "middleware")
		handler := Recovery(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		body := w.Body.String()
		if body != "Internal Server Error\n" {
			t.Errorf("Expected body 'Internal Server Error', got '%s'", body)
		}
	})

	t.Run("TestNormalRequest", func(t *testing.T) {
		log := logger.GetDefaultLogger().WithField("test", "middleware")
		called := false
		handler := Recovery(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if !called {
			t.Error("Next handler should be called for normal request")
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestResponseWriter(t *testing.T) {
	t.Run("TestResponseWriterStatusCode", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		rw.WriteHeader(http.StatusNotFound)
		if rw.statusCode != http.StatusNotFound {
			t.Errorf("Expected status code 404, got %d", rw.statusCode)
		}

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected underlying writer status code 404, got %d", w.Code)
		}
	})

	t.Run("TestResponseWriterDefaultStatusCode", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		if rw.statusCode != http.StatusOK {
			t.Errorf("Expected default status code 200, got %d", rw.statusCode)
		}
	})
}
