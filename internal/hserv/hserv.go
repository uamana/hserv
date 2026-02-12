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
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	RootDir      string
	SidName      string
	UidName      string
	ChunkExt     string
	ChunkMIME    string
	BufferSize   int
	TLSCertPath  string
	TLSKeyPath   string
	ChunkWriter  *chunklog.Writer
}

func (h *HServ) Run(ctx context.Context) (err error) {
	h.RootDir, err = filepath.Abs(h.RootDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of root directory: %w", err)
	}

	if h.ChunkMIME == "" {
		h.ChunkMIME = mime.TypeByExtension(h.ChunkExt)
	}

	kpr, err := NewKeypairReloader(h.TLSCertPath, h.TLSKeyPath)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:         h.Addr,
		ReadTimeout:  h.ReadTimeout,
		WriteTimeout: h.WriteTimeout,
		Handler:      http.HandlerFunc(h.handler),
		TLSConfig:    &tls.Config{GetCertificate: kpr.GetCertificateFunc()},
	}

	slog.Info("hserv",
		"addr", h.Addr,
		"rootDir", h.RootDir,
		"sidName", h.SidName,
		"chunkExt", h.ChunkExt,
		"chunkMIME", h.ChunkMIME,
		"bufferSize", h.BufferSize,
		"tlsCertPath", h.TLSCertPath,
		"tlsKeyPath", h.TLSKeyPath,
	)

	// Graceful shutdown: wait for SIGINT/SIGTERM (or parent context cancellation),
	// then drain chunk writer before shutting down the HTTP server.
	srvCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServeTLS("", "")
	}()

	select {
	case err := <-errCh:
		// Server exited on its own (listener error or similar).
		// Best-effort flush of any pending chunklog events.
		if h.ChunkWriter != nil {
			shutdownCtx, cancel := context.WithTimeout(srvCtx, 5*time.Second)
			defer cancel()
			h.ChunkWriter.Shutdown(shutdownCtx)
		}
		return err

	case <-srvCtx.Done():
		// OS signal: first drain the chunklog writer, then gracefully
		// shut down the HTTP server.
		shutdownCtx, cancel := context.WithTimeout(srvCtx, 5*time.Second)
		defer cancel()

		if h.ChunkWriter != nil {
			h.ChunkWriter.Shutdown(shutdownCtx)
		}

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	}
}
