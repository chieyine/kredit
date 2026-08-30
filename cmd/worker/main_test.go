package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandlerExposesLivenessAndReadiness(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		ready      func() error
		wantStatus int
		wantBody   string
	}{
		{name: "live", path: "/healthz", ready: func() error { return errors.New("database unavailable") }, wantStatus: http.StatusOK, wantBody: "ok"},
		{name: "ready", path: "/readyz", ready: func() error { return nil }, wantStatus: http.StatusOK, wantBody: "ok"},
		{name: "not ready", path: "/readyz", ready: func() error { return errors.New("database unavailable") }, wantStatus: http.StatusServiceUnavailable, wantBody: "not ready"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			response := httptest.NewRecorder()

			healthHandler(tt.ready).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if response.Body.String() != tt.wantBody {
				t.Fatalf("body = %q, want %q", response.Body.String(), tt.wantBody)
			}
			if response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("health response must not be cached")
			}
		})
	}
}

func TestHealthHandlerRejectsMutationMethods(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/readyz", nil)
	response := httptest.NewRecorder()

	healthHandler(func() error { return nil }).ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
