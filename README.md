# hserv

HTTP server with TLS support for serving HLS playlists (`.m3u8`) and chunks (`.ts`) for [radiostream](https://github.com/uamana/radiostream).

Main purpose is to store session and user info in TimescaleDB for statistics.

## Chunk name format

To properly handle DB updates chunk name **MUST** follow this format:
```
<codec>_<quality>_<timestamp>_<duration>_<sequence>.<ext>
```

Where:
| Name | Description |
|------|-------------|
| `codec` | Audio codec (mp3, acc, etc.) |
| `quality` | HLS stream quality: `lofi`, `hifi`, `midfi` |
| `timestamp` | Unix time (timestamp) of chunk creation |
| `sequence` | Sequnce number (may be zero), now not used |
| `ext` | Extension of chunk file, value of `-ext` command line arg |

## Usage

```bash
hserv -addr :6443 -root /path/to/content -cert server.crt -key server.key
```

## Arguments

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:6443` | Address to listen on |
| `-root` | `.` | Root directory to serve |
| `-sid` | `sid` | Name of the session ID query parameter |
| `-uid` | `uid` | Name of the user ID cookie |
| `-ext` | `.ts` | Extension of chunk files |
| `-mime` | `video/mp2t` | MIME type of chunk files |
| `-bsize` | `1024` | Buffer size for playlist scanner |
| `-cert` | — | Path to TLS certificate |
| `-key` | — | Path to TLS private key |
| `-db` | — | Connection string for the TimescaleDB database (enables chunk logging) |
| `-workers` | `0` | Number of workers for the chunk log writer (`0` = number of CPU cores) |
| `-batch` | `1000` | Batch size (rows per write) for the chunk log writer |
| `-batchtimeout` | `200ms` | Maximum time to wait before flushing a partial batch |
| `-channelcap` | `0` | Capacity of the chunk event channel (`0` = auto: workers × batch × 2) |



