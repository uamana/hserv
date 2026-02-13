# hserv

HTTP server with TLS support for serving HLS playlists (`.m3u8`) and chunks (`.ts`) for [radiostream](https://github.com/uamana/radiostream).

Main purpose is to store session and user info in TimescaleDB for statistics.

## Chunk name format

To properly handle DB updates chunk name **MUST** follow this format:
```
<mount>/<codec>_<quality>_<timestamp>_<duration>_<sequence>.<ext>
```

Where:
| Name | Description |
|------|-------------|
| `mount` | Mount (radio name, multiple HLS radios supported) |
| `codec` | Audio codec (mp3, aac, etc.) |
| `quality` | HLS stream quality: `lofi`, `hifi`, `midfi` |
| `timestamp` | Unix time (timestamp) of chunk creation |
| `duration` | Duration of chunk in seconds, float |
| `sequence` | Sequnce number (may be zero), now not used |
| `ext` | Extension of chunk file, value of `-ext` command line arg |

## Usage

```bash
hserv -addr :6443 -root /path/to/content -cert server.crt -key server.key
```

## Command line flags

| Flag | Default | Description | Docker env var |
|------|---------|-------------|----------------|
| `-addr` | `:6443` | Address to listen on | `HSERV_ADDR` |
| `-root` | `.` | Root directory to serve | `HSERV_ROOT` |
| `-sid` | `sid` | Name of the session ID query parameter | `HSERV_SID` |
| `-uid` | `uid` | Name of the user ID cookie | `HSERV_UID` |
| `-ext` | `.ts` | Extension of chunk files | `HSERV_EXT` |
| `-mime` | `video/mp2t` | MIME type of chunk files | `HSERV_MIME` |
| `-bsize` | `1024` | Buffer size for playlist scanner | `HSERV_BSIZE` |
| `-read-timeout` | `5s` | HTTP read timeout | `HSERV_READ_TIMEOUT` |
| `-write-timeout` | `5s` | HTTP write timeout | `HSERV_WRITE_TIMEOUT` |
| `-tls` | `true` | Enable TLS (requires `-cert` and `-key`) | `HSERV_TLS` |
| `-cert` | — | Path to TLS certificate | `HSERV_CERT` |
| `-key` | — | Path to TLS private key | `HSERV_KEY` |
| `-db` | — | Connection string for the database (enables session tracking) | `HSERV_DB` |
| `-session-timeout` | `60s` | Inactivity timeout before a session is flushed to the database | `HSERV_SESSION_TIMEOUT` |
| `-icecast-timeout` | `24h` | Inactivity timeout before an Icecast session is flushed to the database | `HSERV_ICECAST_TIMEOUT` |
| `-channelcap` | `10000` | Channel capacity for the session tracker | `HSERV_CHANNELCAP` |
| `-reaper` | `10s` | Interval for the session reaper | `HSERV_REAPER` |
