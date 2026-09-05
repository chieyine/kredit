package main

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestRunSelfHealthcheck(t *testing.T) {
	healthyClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != selfHealthcheckURL {
			t.Fatalf("healthcheck destination changed: got %q want %q", request.URL.String(), selfHealthcheckURL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
			Request:    request,
		}, nil
	})}

	if code := runSelfHealthcheckWithClient(healthyClient); code != 0 {
		t.Fatalf("expected healthcheck code 0 for healthy server, got %d", code)
	}

	unhealthyClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})}
	if code := runSelfHealthcheckWithClient(unhealthyClient); code != 1 {
		t.Fatalf("expected healthcheck code 1 for unreachable server, got %d", code)
	}

	if code := runSelfHealthcheckWithClient(nil); code != 1 {
		t.Fatalf("expected healthcheck code 1 for nil client, got %d", code)
	}
}
