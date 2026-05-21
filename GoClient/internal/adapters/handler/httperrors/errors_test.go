package httperrors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"weicloth/internal/core/apperrors"

	"github.com/gin-gonic/gin"
)

func TestWriteServiceError_InvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteServiceError(c, fmt.Errorf("login failed: %w", apperrors.ErrInvalidCredentials))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want %d", w.Code, http.StatusUnauthorized)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "invalid credentials" {
		t.Fatalf("body: %+v", body)
	}
}

func TestWriteServiceError_InternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteServiceError(c, errors.New("pq: connection refused"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", w.Code, http.StatusInternalServerError)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "internal server error" {
		t.Fatalf("body: %+v", body)
	}
	if body["error"] == "pq: connection refused" {
		t.Fatal("leaked internal error")
	}
}

func TestWriteServiceError_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteServiceError(c, &apperrors.ValidationError{Field: "status", Message: "invalid status"})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want %d", w.Code, http.StatusBadRequest)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "invalid status" {
		t.Fatalf("body: %+v", body)
	}
}
