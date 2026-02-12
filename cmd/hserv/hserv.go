package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/uamana/hserv/internal/chunklog"
	"github.com/uamana/hserv/internal/hserv"
)

func main() {
	var (
		addr           string
		rootDir        string
		sidName        string
		uidName        string
		chunkExt       string
		chunkMIME      string
		bufferSize     int
		tlsCertPath    string
		tlsKeyPath     string
		dbConnString   string
		sessionTimeout time.Duration
		channelCap     int
	)
	flag.StringVar(&addr, "addr", ":6443", "address to listen on")
	flag.StringVar(&rootDir, "root", ".", "root directory to serve")
	flag.StringVar(&sidName, "sid", "sid", "name of the sid parameter")
	flag.StringVar(&uidName, "uid", "uid", "name of the uid cookie")
	flag.StringVar(&chunkExt, "ext", ".ts", "extension of the chunk files")
	flag.StringVar(&chunkMIME, "mime", "video/mp2t", "MIME type of the chunk files")
	flag.IntVar(&bufferSize, "bsize", 1024, "buffer size for the scanner")
	flag.StringVar(&tlsCertPath, "cert", "", "path to the TLS certificate")
	flag.StringVar(&tlsKeyPath, "key", "", "path to the TLS key")
	flag.StringVar(&dbConnString, "db", "", "connection string for the database")
	flag.DurationVar(&sessionTimeout, "session-timeout", 60*time.Second, "inactivity timeout before a session is flushed to the database")
	flag.IntVar(&channelCap, "channelcap", 10000, "channel capacity for the session tracker")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := &hserv.HServ{
		Addr:        addr,
		RootDir:     rootDir,
		SidName:     sidName,
		UidName:     uidName,
		ChunkExt:    chunkExt,
		ChunkMIME:   chunkMIME,
		BufferSize:  bufferSize,
		TLSCertPath: tlsCertPath,
		TLSKeyPath:  tlsKeyPath,
	}

	if dbConnString != "" {
		tracker, err := chunklog.NewSessionTracker(ctx, chunklog.Config{
			ConnString:     dbConnString,
			SessionTimeout: sessionTimeout,
			ChannelCap:     channelCap,
		})
		if err != nil {
			slog.Error("failed to create session tracker", "error", err)
			os.Exit(1)
		}
		h.SessionTracker = tracker
	}

	if err := h.Run(ctx); err != nil {
		slog.Error("failed to run hserv", "error", err)
		os.Exit(1)
	}
}
