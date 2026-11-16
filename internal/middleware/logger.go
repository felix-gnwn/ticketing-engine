package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriterDelegator struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func newResponseWriterDelegator(w http.ResponseWriter) *responseWriterDelegator {
	return &responseWriterDelegator{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriterDelegator) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterDelegator) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)

	return n, err
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := newResponseWriterDelegator(w)

		next.ServeHTTP(wrapped, r)
		duration := time.Since(start)

		slog.InfoContext(r.Context(), "HTTP Request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrapped.statusCode),
			slog.Duration("latency", duration),
			slog.Int64("bytes_written", wrapped.written),
			slog.String("client_ip", r.RemoteAddr),
		)
	})
}
