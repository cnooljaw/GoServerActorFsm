package main

import (
	"log"
	"log/slog"

	"goserveractorfsm/internal/config"
	"goserveractorfsm/internal/logx"
	"goserveractorfsm/internal/ws"
)

func main() {
	cfg := config.Default()
	logger := logx.Default()
	server := ws.NewServerWithLogger(cfg, logger)

	logger.Info("server_starting",
		slog.Int("port", cfg.Port),
		slog.String("websocket", "ws://127.0.0.1"+server.Addr()+"/ws"),
	)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
