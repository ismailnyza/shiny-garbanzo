package websocket

import (
	"net/http"
	"testing"
)

func TestCheckOriginAllowsConfiguredFrontend(t *testing.T) {
	h := NewWSHandler(NewHub(), nil, "https://app.example.com", "production")
	req, _ := http.NewRequest(http.MethodGet, "/ws/sessions/token", nil)
	req.Header.Set("Origin", "https://app.example.com")

	if !h.checkOrigin(req) {
		t.Fatal("expected configured frontend origin to be allowed")
	}
}

func TestCheckOriginRejectsMissingOrForeignOrigin(t *testing.T) {
	h := NewWSHandler(NewHub(), nil, "https://app.example.com", "production")

	missing, _ := http.NewRequest(http.MethodGet, "/ws/sessions/token", nil)
	if h.checkOrigin(missing) {
		t.Fatal("expected missing origin to be rejected")
	}

	foreign, _ := http.NewRequest(http.MethodGet, "/ws/sessions/token", nil)
	foreign.Header.Set("Origin", "https://evil.example.com")
	if h.checkOrigin(foreign) {
		t.Fatal("expected foreign origin to be rejected")
	}
}

func TestCheckOriginAllowsAllOriginsInDevelopment(t *testing.T) {
	h := NewWSHandler(NewHub(), nil, "", "development")

	missing, _ := http.NewRequest(http.MethodGet, "/ws/sessions/token", nil)
	if !h.checkOrigin(missing) {
		t.Fatal("expected development mode to allow missing origin")
	}

	foreign, _ := http.NewRequest(http.MethodGet, "/ws/sessions/token", nil)
	foreign.Header.Set("Origin", "https://evil.example.com")
	if !h.checkOrigin(foreign) {
		t.Fatal("expected development mode to allow foreign origin")
	}
}
