package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/your-org/arx-ca/internal/logging"
)

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Logger returns middleware that logs each HTTP request when the log level is debug.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(recorder, r)

		if !logging.IsDebug() {
			return
		}

		duration := time.Since(start)
		logging.Logger().Debug("http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.statusCode),
			slog.Duration("duration", duration),
			slog.String("remote", r.RemoteAddr),
		)
	})
}
