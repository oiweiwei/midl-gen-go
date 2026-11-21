// Package logx carries an *slog.Logger in a context.Context, so that library
// code deep in a call chain can log without every signature taking a logger.
package logx

import (
	"context"
	"io"
	"log/slog"
)

type ctxKey struct{}

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

// With returns a context that carries the logger.
func With(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// From returns the logger carried by the context, or a logger that discards
// everything; it never returns nil.
func From(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return discard
}
