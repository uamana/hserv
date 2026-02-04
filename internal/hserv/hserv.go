package hserv

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"time"
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
}

func (h *HServ) Run() (err error) {
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

	return srv.ListenAndServeTLS("", "")
}
