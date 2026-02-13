package chunklog

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	useragent "github.com/medama-io/go-useragent"
)

// Config holds SessionTracker configuration.
type Config struct {
	ChannelCap     int
	SessionTimeout time.Duration
	ConnString     string
	ReaperInterval time.Duration
}

// SessionTracker tracks active sessions in memory and flushes completed
// sessions to the database when they become idle.
type SessionTracker struct {
	events         chan ChunkEvent
	pool           *pgxpool.Pool
	wg             sync.WaitGroup
	once           sync.Once
	drops          atomic.Uint64
	flushErrors    atomic.Uint64
	timeout        time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
	reaperInterval time.Duration
}

// NewSessionTracker creates a new tracker, connects to the database, and
// starts the background goroutine.
func NewSessionTracker(ctx context.Context, cfg Config) (*SessionTracker, error) {
	// pool, err := pgxpool.New(ctx, cfg.ConnString)
	// if err != nil {
	// 	return nil, err
	// }

	poolCfg, err := pgxpool.ParseConfig(cfg.ConnString)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	events := make(chan ChunkEvent, cfg.ChannelCap)
	ctx, cancel := context.WithCancel(ctx)
	t := &SessionTracker{
		events:         events,
		pool:           pool,
		timeout:        cfg.SessionTimeout,
		ctx:            ctx,
		cancel:         cancel,
		reaperInterval: cfg.ReaperInterval,
	}

	t.wg.Add(1)
	go t.run()

	return t, nil
}

// Send enqueues a chunk event. Non-blocking; returns false if the channel
// is full (event dropped).
func (t *SessionTracker) Send(e ChunkEvent) bool {
	select {
	case t.events <- e:
		return true
	default:
		t.drops.Add(1)
		return false
	}
}

// Shutdown closes the event channel, waits for the background goroutine to
// drain and flush all remaining sessions, then closes the database pool.
func (t *SessionTracker) Shutdown(ctx context.Context) {
	t.once.Do(func() {
		close(t.events)
	})

	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}

	t.cancel()
	t.pool.Close()
}

// Drops returns the number of events dropped due to a full channel.
func (t *SessionTracker) Drops() uint64 {
	return t.drops.Load()
}

// run is the single goroutine that owns the sessions map.
func (t *SessionTracker) run() {
	defer t.wg.Done()
	sessions := make(map[uuid.UUID]*Session)
	parser := useragent.NewParser()
	reaper := time.NewTicker(t.reaperInterval)
	defer reaper.Stop()

	for {
		select {
		case e, ok := <-t.events:
			if !ok {
				// Channel closed — flush all remaining sessions.
				t.flushAll(sessions)
				return
			}
			t.handleEvent(e, sessions, parser)

		case <-reaper.C:
			t.reap(sessions)

		case <-t.ctx.Done():
			t.flushAll(sessions)
			return
		}
	}
}

func (t *SessionTracker) handleEvent(e ChunkEvent, sessions map[uuid.UUID]*Session, parser *useragent.Parser) {
	sid, err := uuid.Parse(e.SID)
	if err != nil {
		sid = uuid.Nil
	}

	// TODO: do we need to store uuid.Nil sessions?
	// TODO: handle icecast session with icecast to uiid mapping

	if s, ok := sessions[sid]; ok {
		s.LastActive = e.Time
		s.TotalBytes += e.ChunkSize
		return
	}

	sessions[sid] = newSessionFromEvent(&e, parser)
}

func (t *SessionTracker) reap(sessions map[uuid.UUID]*Session) {
	now := time.Now()
	var expired []*Session
	for sid, s := range sessions {
		if now.Sub(s.LastActive) > t.timeout {
			s.Duration = now.Sub(s.StartTime)
			expired = append(expired, s)
			delete(sessions, sid)
		}
	}

	if err := t.addListenersTotal(len(expired)); err != nil {
		t.flushErrors.Add(1)
		slog.Error("failed to add listeners total", "error", err, "count", len(expired))
	}

	if len(expired) > 0 {
		if err := t.flushSessions(expired); err != nil {
			t.flushErrors.Add(1)
			slog.Error("failed to flush expired sessions", "error", err,
				"count", len(expired))
		}
	}
}

func (t *SessionTracker) flushAll(sessions map[uuid.UUID]*Session) {
	if len(sessions) == 0 {
		return
	}
	all := make([]*Session, 0, len(sessions))
	for _, s := range sessions {
		all = append(all, s)
	}
	if err := t.flushSessions(all); err != nil {
		t.flushErrors.Add(1)
		slog.Error("failed to flush sessions on shutdown", "error", err, "count", len(all))
	}
}

func (t *SessionTracker) flushSessions(sessions []*Session) error {
	if len(sessions) == 0 {
		return nil
	}
	conn, err := t.pool.Acquire(t.ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	rows := make([][]interface{}, 0, len(sessions))
	for _, s := range sessions {
		rows = append(rows, s.row())
	}

	_, err = conn.Conn().CopyFrom(
		t.ctx,
		pgx.Identifier{"sessions"},
		sessionColumns,
		pgx.CopyFromRows(rows),
	)

	return err
}

func (t *SessionTracker) addListenersTotal(total int) error {
	conn, err := t.pool.Acquire(t.ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Conn().Exec(
		t.ctx,
		"INSERT INTO listeners_total (timestamp, count) VALUES ($1, $2, $3)",
		time.Now(),
		total,
	)

	return err
}
