package common

import (
	"log/slog"
	"os"

	"github.com/sectrean/di-kit"
)

var DependencyModule = di.Module{
	di.WithService(NewLogger()),
	//...
}

func NewLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}
