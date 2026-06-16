package logx

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewWritesStructuredTextLog(t *testing.T) {
	var out bytes.Buffer
	logger := New(&out)

	logger.Info("client_connected", slog.String("remote", "127.0.0.1:9000"))

	got := out.String()
	if !strings.Contains(got, "level=INFO") {
		t.Fatalf("log = %q, want level=INFO", got)
	}
	if !strings.Contains(got, "msg=client_connected") {
		t.Fatalf("log = %q, want msg=client_connected", got)
	}
	if !strings.Contains(got, "remote=127.0.0.1:9000") {
		t.Fatalf("log = %q, want remote field", got)
	}
}
