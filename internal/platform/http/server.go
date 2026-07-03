// Package httpserver hosts the HTTP layer: router, middleware, and server
// lifecycle. Handlers contain no business logic — they decode, delegate to
// application services, and encode.
package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Serve runs the HTTP server until SIGINT/SIGTERM, then shuts down
// gracefully.
func Serve(log *slog.Logger, port string, handler http.Handler) error {
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		// No global write timeout: the SSE stream is long-lived.
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", "port", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-stop:
		log.Info("shutting down", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	}
}
