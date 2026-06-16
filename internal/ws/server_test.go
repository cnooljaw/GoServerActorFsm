package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"goserveractorfsm/internal/config"
)

func TestNewServerUsesConfiguredPort(t *testing.T) {
	server := NewServer(config.ServerConfig{Port: 9000})

	if server.Addr() != ":9000" {
		t.Fatalf("Addr() = %q, want :9000", server.Addr())
	}
}

func TestNewServerRoutesWebSocketPath(t *testing.T) {
	server := NewServer(config.ServerConfig{Port: 9000})
	req := httptest.NewRequest(http.MethodGet, "/not-ws", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
