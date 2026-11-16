package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

const (
	TraceIDKey     contextKey = "traceID"
	HeaderXTraceID            = "X-Trace-ID"
)

type TraceHandler struct {
	slog.Handler
}

func NewTraceHandler(h slog.Handler) *TraceHandler {
	return &TraceHandler{
		Handler: h,
	}
}

func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	if traceID := GetTraceID(ctx); traceID != "" {
		r.AddAttrs(slog.String("trace_id", traceID))
	} else {
		r.AddAttrs(slog.String("trace_id", "unknown"))
	}

	return h.Handler.Handle(ctx, r)
}

func (h *TraceHandler) WithAttr(attrs []slog.Attr) slog.Handler {
	return &TraceHandler{
		Handler: h.Handler.WithAttrs(attrs),
	}
}

func (h *TraceHandler) WithGroup(name string) slog.Handler {
	return &TraceHandler{
		Handler: h.Handler.WithGroup(name),
	}
}

func TraceID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(HeaderXTraceID)
		if traceID == "" {
			traceID = uuid.New().String()
		}

		w.Header().Set(HeaderXTraceID, traceID)

		ctx := context.WithValue(r.Context(), TraceIDKey, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}

	return ""
}
