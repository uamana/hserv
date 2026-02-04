# hserv

HTTP server with TLS support for serving HLS playlists (`.m3u8`) and chunks (`.ts`) for [radiostream](https://github.com/uamana/radiostream).

Main purpose is to store session and user info in TimescaleDB for statistics.

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
