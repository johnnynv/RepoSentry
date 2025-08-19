package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewJSONResponse(t *testing.T) {
	data := map[string]string{"test": "data"}
	response := NewJSONResponse(data)

	if !response.Success {
		t.Error("Expected Success to be true")
	}

	if response.Data == nil {
		t.Error("Expected data to be set")
	}

	if response.Error != "" {
		t.Error("Expected error to be empty")
	}

	if response.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestNewErrorResponse(t *testing.T) {
	errorMsg := "test error"
	response := NewErrorResponse(errorMsg)

	if response.Success {
		t.Error("Expected Success to be false")
	}

	if response.Data != nil {
		t.Error("Expected data to be nil")
	}

	if response.Error != errorMsg {
		t.Errorf("Expected error to be %s, got %s", errorMsg, response.Error)
	}

	if response.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestResponse_Write(t *testing.T) {
	// Test successful response
	data := map[string]string{"test": "data"}
	response := NewJSONResponse(data)

	recorder := httptest.NewRecorder()
	response.Write(recorder)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse response body
	var respData map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &respData)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if respData["success"] != true {
		t.Error("Expected success to be true")
	}

	// Test error response
	errorResponse := NewErrorResponse("test error")
	recorder = httptest.NewRecorder()
	errorResponse.Write(recorder)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", recorder.Code)
	}
}

func TestResponse_WriteWithStatus(t *testing.T) {
	response := NewJSONResponse("test")
	recorder := httptest.NewRecorder()

	response.WriteWithStatus(recorder, http.StatusCreated)

	if recorder.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", recorder.Code)
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestResponse_JSONSerialization(t *testing.T) {
	now := time.Now()
	response := &Response{
		Success:   true,
		Data:      map[string]string{"key": "value"},
		Error:     "",
		Timestamp: now,
	}

	recorder := httptest.NewRecorder()
	response.Write(recorder)

	var parsed Response
	err := json.Unmarshal(recorder.Body.Bytes(), &parsed)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if parsed.Success != response.Success {
		t.Error("Success field not serialized correctly")
	}

	if parsed.Error != response.Error {
		t.Error("Error field not serialized correctly")
	}

	// Note: timestamp comparison might have slight differences due to JSON serialization
	if parsed.Timestamp.Unix() != response.Timestamp.Unix() {
		t.Error("Timestamp not serialized correctly")
	}
}
