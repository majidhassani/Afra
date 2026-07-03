package httpserver

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"casemind/internal/auth"
	"casemind/internal/privacy"
	apperrors "casemind/pkg/errors"
	"casemind/pkg/response"
)

type Middleware func(http.Handler) http.Handler

func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// Recover converts panics into 500 responses.
func Recover(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered", "panic", rec, "path", r.URL.Path)
					response.Err(w, apperrors.Internal(fmt.Errorf("panic: %v", rec), "internal server error"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogger logs each request with latency.
func RequestLogger(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			log.Info("http request",
				"method", r.Method, "path", r.URL.Path,
				"status", sw.status, "duration_ms", time.Since(start).Milliseconds())
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// CORS allows the mobile client and local tooling to call the API.
func CORS() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit enforces a fixed-window per-key limit in Redis. It fails open
// when Redis is unreachable (availability over strictness for a game API).
func RateLimit(rdb *goredis.Client, log *slog.Logger, limit int, window time.Duration, scope string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := fmt.Sprintf("ratelimit:%s:%s:%d", scope, clientKey(r), time.Now().Unix()/int64(window.Seconds()))
			count, err := rdb.Incr(r.Context(), key).Result()
			if err != nil {
				log.Warn("rate limit unavailable", "error", err)
				next.ServeHTTP(w, r)
				return
			}
			if count == 1 {
				rdb.Expire(r.Context(), key, window)
			}
			if count > int64(limit) {
				response.Err(w, apperrors.RateLimited("too many requests, slow down"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientKey prefers the authenticated user, falling back to client IP.
func clientKey(r *http.Request) string {
	if userID, ok := auth.UserID(r.Context()); ok {
		return "u:" + userID.String()
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return "ip:" + host
}

// PrivacyGuard is the last line of defense for the Truth Layer: it buffers
// JSON responses and refuses to send any body containing forbidden fields.
// Non-JSON responses (SSE stream) pass through untouched.
func PrivacyGuard(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gw := &guardWriter{rw: w}
			next.ServeHTTP(gw, r)
			if !gw.buffering {
				return
			}
			body := gw.buf.Bytes()
			if leaks := privacy.ScanJSON(body); len(leaks) > 0 {
				log.Error("PRIVACY VIOLATION BLOCKED", "path", r.URL.Path, "fields", leaks)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":{"code":"internal_error","message":"internal server error"}}`))
				return
			}
			if gw.status != 0 {
				w.WriteHeader(gw.status)
			}
			_, _ = w.Write(body)
		})
	}
}

// guardWriter buffers JSON responses; streams everything else through.
type guardWriter struct {
	rw        http.ResponseWriter
	buf       bytes.Buffer
	status    int
	decided   bool
	buffering bool
}

func (g *guardWriter) Header() http.Header { return g.rw.Header() }

func (g *guardWriter) decide() {
	if g.decided {
		return
	}
	g.decided = true
	ct := g.rw.Header().Get("Content-Type")
	g.buffering = ct == "" || strings.HasPrefix(ct, "application/json")
}

func (g *guardWriter) WriteHeader(status int) {
	g.decide()
	if g.buffering {
		g.status = status
		return
	}
	g.rw.WriteHeader(status)
}

func (g *guardWriter) Write(b []byte) (int, error) {
	g.decide()
	if g.buffering {
		return g.buf.Write(b)
	}
	return g.rw.Write(b)
}

func (g *guardWriter) Flush() {
	if g.buffering {
		return
	}
	if f, ok := g.rw.(http.Flusher); ok {
		f.Flush()
	}
}
