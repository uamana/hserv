package hserv

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// KeypairReloader watches for SIGHUP and reloads the TLS certificate
// and key from disk.
type KeypairReloader struct {
	certMu   sync.RWMutex
	cert     *tls.Certificate
	certPath string
	keyPath  string
}

// NewKeypairReloader loads the initial certificate and starts a background
// goroutine that reloads the certificate on SIGHUP.
func NewKeypairReloader(ctx context.Context, certPath, keyPath string) (*KeypairReloader, error) {
	result := &KeypairReloader{
		certPath: certPath,
		keyPath:  keyPath,
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	result.cert = &cert
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP)
		defer signal.Stop(c)
		for {
			select {
			case <-c:
				slog.Info("received SIGHUP, reloading TLS certificate and key")
				if err := result.maybeReload(); err != nil {
					slog.Error("keeping old TLS certificate because the new one could not be loaded", "error", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return result, nil
}

func (kpr *KeypairReloader) maybeReload() error {
	newCert, err := tls.LoadX509KeyPair(kpr.certPath, kpr.keyPath)
	if err != nil {
		return err
	}
	kpr.certMu.Lock()
	defer kpr.certMu.Unlock()
	kpr.cert = &newCert
	return nil
}

func (kpr *KeypairReloader) GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		kpr.certMu.RLock()
		defer kpr.certMu.RUnlock()
		return kpr.cert, nil
	}
}
