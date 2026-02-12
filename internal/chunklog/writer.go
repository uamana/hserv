package chunklog

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds Writer configuration.
type Config struct {
	ChannelCap   int
	WorkerCount  int
	BatchSize    int
	BatchTimeout time.Duration
	ConnString   string
}

// Writer consumes chunk events from a buffered channel via worker goroutines.
type Writer struct {
	events      chan ChunkEvent
	pool        *pgxpool.Pool
	wg          sync.WaitGroup
	once        sync.Once
	drops       atomic.Uint64
	flushErrors atomic.Uint64
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewWriter starts the worker pool and returns a Writer.
// pgxpool is initialized here; conn string comes from config.
func NewWriter(ctx context.Context, cfg Config) (*Writer, error) {
	pool, err := pgxpool.New(ctx, cfg.ConnString)
	if err != nil {
		return nil, err
	}

	events := make(chan ChunkEvent, cfg.ChannelCap)
	ctx, cancel := context.WithCancel(ctx)
	w := &Writer{events: events, pool: pool, ctx: ctx, cancel: cancel}

	for i := 0; i < cfg.WorkerCount; i++ {
		w.wg.Add(1)
		go w.worker(cfg, i)
	}

	return w, nil
}

// Send enqueues an event. Non-blocking; returns false if channel is full (event dropped).
func (w *Writer) Send(e ChunkEvent) bool {
	select {
	case w.events <- e:
		return true
	default:
		w.drops.Add(1)
		return false
	}
}

// Shutdown closes the channel, waits for workers to drain, and closes the pgxpool.
// The internal context is cancelled only after workers finish (or the deadline
// expires), so that final flush operations can still reach the database.
func (w *Writer) Shutdown(ctx context.Context) {
	w.once.Do(func() {
		close(w.events)
	})

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}

	w.cancel()
	w.pool.Close()
}

// Drops returns the number of events dropped due to channel full.
func (w *Writer) Drops() uint64 {
	return w.drops.Load()
}

func (w *Writer) flush(batch *BatchBuffer) error {
	if batch.Len() == 0 {
		return nil
	}
	conn, err := w.pool.Acquire(w.ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	_, err = conn.Conn().CopyFrom(
		w.ctx,
		pgx.Identifier{"chunk_requests"},
		chunkRequestColumns,
		batch,
	)
	return err
}

func (w *Writer) worker(cfg Config, id int) {
	defer w.wg.Done()
	logger := slog.With("worker", id)
	batch := NewBatchBuffer(cfg.BatchSize)
	timer := time.NewTimer(cfg.BatchTimeout)
	timer.Stop()
	defer timer.Stop()

	var (
		err       error
		needFlush bool
	)

	for {
		if batch.IsFull() || needFlush {
			if err = w.flush(batch); err != nil {
				w.flushErrors.Add(1)
				logger.Error("failed to flush batch", "error", err, "total errors", w.flushErrors.Load())
			}
			timer.Stop()
			batch.Reset()
			needFlush = false
			continue
		}
		select {
		case e, ok := <-w.events:
			if !ok {
				if err = w.flush(batch); err != nil {
					w.flushErrors.Add(1)
					logger.Error("failed to flush batch on shutdown", "error", err, "total errors", w.flushErrors.Load())
				}
				return
			}
			batch.Add(e)
			if batch.Len() == 1 {
				timer.Reset(cfg.BatchTimeout)
			}
		case <-timer.C:
			needFlush = true
		case <-w.ctx.Done():
			return
		}
	}
}
