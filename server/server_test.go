package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateValidation(t *testing.T) {
	h := Router()
	req := httptest.NewRequest("GET", "https://ddns.c6e.me/update/c6e.me/hunter2/127.0.0.1", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	status := w.Result().StatusCode
	if status != http.StatusOK {
		t.Errorf("valid no-paramater requested, expected OK, got %v", status)
	}
}
