package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/uamana/hserv/internal/hserv"
)

func main() {
	var (
		addr        string
		rootDir     string
		sidName     string
		uidName     string
		chunkExt    string
		chunkMIME   string
		bufferSize  int
		tlsCertPath string
		tlsKeyPath  string
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
	flag.Parse()

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

	if err := hserv.Run(); err != nil {
		slog.Error("failed to run hserv", "error", err)
		os.Exit(1)
	}
}
