package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/uamana/hserv/internal/chunklog"
	"github.com/uamana/hserv/internal/hserv"
)

func main() {
	var (
		addr         string
		rootDir      string
		sidName      string
		uidName      string
		chunkExt     string
		chunkMIME    string
		bufferSize   int
		tlsCertPath  string
		tlsKeyPath   string
		dbConnString string
		workerCount  int
		batchSize    int
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
	flag.IntVar(&workerCount, "workers", 0, "number of workers for the chunk log writer")
	flag.IntVar(&batchSize, "batch", 1000, "batch size for the chunk log writer")

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if dbConnString != "" {
		if workerCount <= 0 {
			slog.Error("worker count must be greater than 0 when database connection string is provided")
			os.Exit(1)
		}
		if batchSize <= 0 {
			slog.Error("batch size must be greater than 0 when database connection string is provided")
			os.Exit(1)
		}
	}

	hserv := &hserv.HServ{
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
		chunkWriter, err := chunklog.NewWriter(ctx, chunklog.Config{
			ConnString:  dbConnString,
			WorkerCount: workerCount,
		})
		if err != nil {
			slog.Error("failed to create chunk log writer", "error", err)
			os.Exit(1)
		}
		hserv.ChunkWriter = chunkWriter
	}

	if err := hserv.Run(ctx); err != nil {
		slog.Error("failed to run hserv", "error", err)
		os.Exit(1)
	}
}
