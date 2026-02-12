# Worker Pool for DB Updates — Implementation Plan

## Overview

Add a separate package with a worker pool that consumes chunk request events from a buffered channel and batch-inserts them into TimescaleDB.

---

## Phase 1: Separate Package

1. **Create `internal/chunklog` package**
  - Owns event struct, channel, worker pool, DB writes.
  - Exports: `ChunkEvent`, `Writer` (or `Pool`), `NewWriter`, `Writer.Send`, `Writer.Shutdown`.
2. **Event struct**
  - Define `ChunkEvent` (time, path, IP, UA, sid, uid).
  - Pass by value (struct is small).
3. **Writer API**
  - `NewWriter(ctx, config)` → starts worker pool, returns `*Writer`.
  - `Writer.Send(ChunkEvent)` → non-blocking send; returns false if channel full (drop).
  - `Writer.Shutdown(ctx)` → closes channel, waits for workers to drain.
4. **Config struct**
  - Channel cap, worker count, batch size, batch timeout, conn string.
  - All configurable via constructor args (caller reads flags/env).

---

## Phase 2: Worker Pool + DB

1. **DB connection**
  - pgxpool inside `chunklog`; init in `NewWriter`, close in `Shutdown`.
  - Conn string from config.
2. **Worker pool**
  - N workers (e.g. 2–4); each runs own goroutine.
  - Shared buffered channel; workers compete for items.
  - Each worker: read from channel, batch (500 rows or 200ms), `INSERT` batch.
  - Use `pgxpool` so each worker gets a connection from the pool.
3. **TimescaleDB table**
  - Hypertable `chunk_requests` with `time` as time column.
  - Columns: `time`, `path`, `ip`,  referer, `user_agent`, `sid`, `uid`.
4. **Ordering**
  - Events may be inserted out of order; acceptable for analytics.

---

## Phase 3: Integration

1. **HServ integration**
  - `HServ` holds optional `*chunklog.Writer` (nil when disabled).
  - In `Run()`, if DB enabled: create `chunklog.Writer`, pass to handler via closure or struct field.
  - Handler: after successful chunk/m3u8 response, call `writer.Send(ChunkEvent)`.
2. **Graceful shutdown**
  - On `HServ` shutdown: call `writer.Shutdown(ctx)` before server shutdown.
3. **Feature flag**
  - Env/flag to disable DB logging; when disabled, `writer` is nil, handler skips send.

---

## Phase 4: Hardening

1. **Retry logic**
  - Exponential backoff on DB errors.
    - Max retries, then log/drop.
2. **Backpressure & metrics**
  - Non-blocking send; drop when full; optional counter for drops.
    - Optional: channel length, insert latency.
3. **Tests**
  - Unit: batch builder, retry logic.
    - Integration: testcontainer TimescaleDB.

---

## File Layout

```
internal/chunklog/
  writer.go       # Writer, NewWriter, Send, Shutdown
  events.go       # ChunkEvent struct
  worker.go       # worker loop, batching, insert

internal/hserv/
  handler.go      # call chunklog.Writer.Send when writer != nil
  hserv.go        # create chunklog.Writer if enabled, pass to handler, Shutdown
```

---

## Dependencies

- `github.com/jackc/pgx/v5` + `pgxpool`

