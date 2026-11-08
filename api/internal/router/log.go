package router

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/markgoho/troop-813/api/internal/contract"
)

// Logger returns l, or a logger that discards when l is nil. Handlers keep
// their *slog.Logger optional (tests and tools leave it unset) and read it
// through this. Log IDs, never Scout names.
func Logger(l *slog.Logger) *slog.Logger {
	if l == nil {
		return slog.New(slog.DiscardHandler)
	}
	return l
}

// InternalError logs err with msg and attrs at error level, then answers 500
// internal. The one place a handler turns an unexpected failure into a response.
func InternalError(ctx context.Context, l *slog.Logger, w http.ResponseWriter, msg string, err error, attrs ...any) {
	Logger(l).ErrorContext(ctx, msg, append([]any{"err", err}, attrs...)...)
	WriteError(w, http.StatusInternalServerError, contract.ErrorCodeInternal)
}
