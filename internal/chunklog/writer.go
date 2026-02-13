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
	ChannelCap            int
	SessionTimeout        time.Duration
	IcecastSessionTimeout time.Duration
	ConnString            string
	ReaperInterval        time.Duration
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
	icecastTimeout time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
	reaperInterval time.Duration
}

// NewSessionTracker creates a new tracker, connects to the database, and
// starts the background goroutine.
func NewSessionTracker(ctx context.Context, cfg Config) (*SessionTracker, error) {
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
		icecastTimeout: cfg.IcecastSessionTimeout,
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
// If ctx expires before the goroutine finishes, the DB context is cancelled
// to unblock any in-flight flush, but Shutdown still waits for the goroutine
// to exit before closing the pool.
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
		// run() exited cleanly after draining and flushing.
	case <-ctx.Done():
		// Deadline hit — cancel the DB context to unblock any stuck flush,
		// then wait for run() to finish before closing the pool.
		t.cancel()
		<-done
	}

	t.cancel()
	t.pool.Close()
}

// Drops returns the number of events dropped due to a full channel.
func (t *SessionTracker) Drops() uint64 {
	return t.drops.Load()
}

// FlushErrors returns the number of failed flush attempts.
func (t *SessionTracker) FlushErrors() uint64 {
	return t.flushErrors.Load()
}

// run is the single goroutine that owns the sessions map.
func (t *SessionTracker) run() {
	defer t.wg.Done()
	sessions := make(map[uuid.UUID]*Session)
	icecastSessions := make(map[int64]uuid.UUID)
	parser := useragent.NewParser()
	reaper := time.NewTicker(t.reaperInterval)
	defer reaper.Stop()
	stats := time.NewTicker(time.Minute)
	defer stats.Stop()

	for {
		select {
		case e, ok := <-t.events:
			if !ok {
				// Channel closed — flush all remaining sessions.
				t.flushAll(sessions)
				return
			}
			t.handleEvent(e, sessions, parser, icecastSessions)

		case <-reaper.C:
			t.reap(sessions, icecastSessions)

		case <-stats.C:
			if err := t.addListenersTotal(sessions); err != nil {
				t.flushErrors.Add(1)
				slog.Error("failed to add listeners total", "error", err)
			}
		}
	}
}

func (t *SessionTracker) handleEvent(e ChunkEvent, sessions map[uuid.UUID]*Session, parser *useragent.Parser, icecastSessions map[int64]uuid.UUID) {
	var (
		sid uuid.UUID
		err error
		ok  bool
	)

	// Icecast use integer ids for sessions, so we need to map them to uuids
	if e.Source == EventSourceIceCast {
		sid, ok = icecastSessions[e.IcecastID]
		if !ok {
			sid = uuid.New()
			icecastSessions[e.IcecastID] = sid
		}
	} else {
		sid, err = uuid.Parse(e.SID)
		if err != nil {
			sid = uuid.Nil
		}
	}

	// TODO: do we need to store uuid.Nil sessions?

	if s, ok := sessions[sid]; ok {
		s.LastActive = e.Time
		s.TotalBytes += e.ChunkSize
		return
	}

	s := newSessionFromEvent(&e, parser)
	s.SID = sid
	sessions[sid] = s
}

func (t *SessionTracker) reap(sessions map[uuid.UUID]*Session, icecastSessions map[int64]uuid.UUID) {
	now := time.Now()
	var expired []*Session
	for sid, s := range sessions {
		if s.Source == EventSourceIceCast {
			// icecast send only two events: start and end, so we need to check
			// if the last active time is after the start time
			if s.LastActive.After(s.StartTime) || now.Sub(s.LastActive) > t.icecastTimeout {
				s.Duration = s.LastActive.Sub(s.StartTime)
				expired = append(expired, s)
				delete(icecastSessions, s.icecastID)
				delete(sessions, sid)
			}
		} else {
			if now.Sub(s.LastActive) > t.timeout {
				s.Duration = s.LastActive.Sub(s.StartTime)
				expired = append(expired, s)
				delete(sessions, sid)
			}
		}
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
		s.Duration = s.LastActive.Sub(s.StartTime)
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

	rows := make([][]any, 0, len(sessions))
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

func (t *SessionTracker) addListenersTotal(sessions map[uuid.UUID]*Session) error {
	hlsTotals := make(map[string]int)
	icecastTotals := make(map[string]int)
	now := time.Now()

	for _, s := range sessions {
		if s.Source == EventSourceHLS {
			if now.Sub(s.LastActive) <= t.timeout {
				hlsTotals[s.Mount]++
			}
		} else {
			if s.StartTime.Equal(s.LastActive) {
				icecastTotals[s.Mount]++
			}
		}
	}

	if len(hlsTotals) == 0 && len(icecastTotals) == 0 {
		return nil
	}

	conn, err := t.pool.Acquire(t.ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	args := make([][]any, len(hlsTotals)+len(icecastTotals))
	var (
		i     int
		mount string
		count int
	)
	for mount, count = range hlsTotals {
		args[i] = []any{now, EventSourceHLS, mount, count}
		i++
	}
	for mount, count = range icecastTotals {
		args[i] = []any{now, EventSourceIceCast, mount, count}
		i++
	}

	_, err = conn.Conn().CopyFrom(
		t.ctx,
		pgx.Identifier{"listeners_total"},
		[]string{"timestamp", "source", "mount", "count"},
		pgx.CopyFromRows(args),
	)

	return err
}
