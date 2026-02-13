package hserv

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/uamana/hserv/internal/chunklog"
)

type HServ struct {
	Addr           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	RootDir        string
	SidName        string
	UidName        string
	ChunkExt       string
	ChunkMIME      string
	BufferSize     int
	TLSEnabled     bool
	TLSCertPath    string
	TLSKeyPath     string
	SessionTracker *chunklog.SessionTracker
}

func (h *HServ) Run(ctx context.Context) (err error) {
	h.RootDir, err = filepath.Abs(h.RootDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of root directory: %w", err)
	}

	if h.ChunkMIME == "" {
		h.ChunkMIME = mime.TypeByExtension(h.ChunkExt)
	}

	srv := &http.Server{
		Addr:         h.Addr,
		ReadTimeout:  h.ReadTimeout,
		WriteTimeout: h.WriteTimeout,
		Handler:      http.HandlerFunc(h.handler),
	}

	if h.TLSEnabled {
		kpr, err := NewKeypairReloader(ctx, h.TLSCertPath, h.TLSKeyPath)
		if err != nil {
			return err
		}
		srv.TLSConfig = &tls.Config{GetCertificate: kpr.GetCertificateFunc()}
	}

	slog.Info("hserv",
		"addr", h.Addr,
		"rootDir", h.RootDir,
		"sidName", h.SidName,
		"chunkExt", h.ChunkExt,
		"chunkMIME", h.ChunkMIME,
		"bufferSize", h.BufferSize,
		"tls", h.TLSEnabled,
	)

	// Graceful shutdown: wait for SIGINT/SIGTERM (or parent context cancellation),
	// then shut down the HTTP server first (drain in-flight requests), followed
	// by the session tracker (flush remaining sessions to DB).
	srvCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if h.TLSEnabled {
			errCh <- srv.ListenAndServeTLS("", "")
		} else {
			errCh <- srv.ListenAndServe()
		}
	}()

	select {
	case err := <-errCh:
		// Server exited on its own (listener error or similar).
		// Best-effort flush of any pending sessions.
		if h.SessionTracker != nil {
			trackerCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			h.SessionTracker.Shutdown(trackerCtx)
		}
		return err

	case <-srvCtx.Done():
		// OS signal or parent context cancelled.
		// First: gracefully shut down the HTTP server to drain in-flight
		// requests so no new Send() calls arrive on the closed channel.
		httpCtx, httpCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer httpCancel()
		if err := srv.Shutdown(httpCtx); err != nil {
			slog.Error("HTTP server shutdown error", "error", err)
		}

		// Second: flush remaining sessions now that no new events can arrive.
		if h.SessionTracker != nil {
			trackerCtx, trackerCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer trackerCancel()
			h.SessionTracker.Shutdown(trackerCtx)
		}
		return err
	}
}

func (h *HServ) handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		slog.Warn("method not allowed", "method", r.Method)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	switch r.URL.Path {
	case "/_in/icecast":
		h.icecastHandler(w, r)
	default:
		h.hlsHandler(w, r)
	}
}
