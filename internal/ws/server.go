package ws

import (
	"fmt"
	"log/slog"
	"net/http"

	"goserveractorfsm/internal/config"
	"goserveractorfsm/internal/logx"
)

type Server struct {
	addr    string
	handler http.Handler
}

func NewServer(cfg config.ServerConfig) *Server {
	return NewServerWithLogger(cfg, logx.Default())
}

func NewServerWithLogger(cfg config.ServerConfig, logger *slog.Logger) *Server {
	mux := http.NewServeMux()
	mux.Handle("/ws", NewHandlerWithLogger(logger))

	return &Server{
		addr:    fmt.Sprintf(":%d", cfg.Port),
		handler: mux,
	}
}

func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) ListenAndServe() error {
	return http.ListenAndServe(s.addr, s.handler)
}
