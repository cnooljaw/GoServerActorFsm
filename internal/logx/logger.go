package logx

import (
	"io"
	"log/slog"
	"os"
)

func New(out io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(out, nil))
}

func Default() *slog.Logger {
	return New(os.Stdout)
}
