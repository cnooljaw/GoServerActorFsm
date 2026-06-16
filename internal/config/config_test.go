package config

import "testing"

func TestDefaultUsesMinimalTeachingProjectConfig(t *testing.T) {
	cfg := Default()

	if cfg.Root != "./" {
		t.Fatalf("Root = %q, want ./", cfg.Root)
	}
	if cfg.Port != 9000 {
		t.Fatalf("Port = %d, want 9000", cfg.Port)
	}
	if cfg.Thread != 2 {
		t.Fatalf("Thread = %d, want 2", cfg.Thread)
	}
	if cfg.Daemon != "" {
		t.Fatalf("Daemon = %q, want empty", cfg.Daemon)
	}
}
