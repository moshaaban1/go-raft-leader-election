package utils

import (
	"os"

	"golang.org/x/exp/slog"
)

func Logger() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
}
