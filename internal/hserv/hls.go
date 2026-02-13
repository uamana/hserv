package hserv

import (
	"bufio"
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uamana/hserv/internal/chunklog"
)

func (h *HServ) hlsHandler(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight.
	if r.Method == http.MethodOptions {
		setHeaders(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := filepath.Join(h.RootDir, filepath.FromSlash(r.URL.Path))
	rel, relErr := filepath.Rel(h.RootDir, path)
	if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		slog.Warn("wrong path", "path", r.URL.Path)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	fileExt := filepath.Ext(path)
	if fileExt != h.ChunkExt && fileExt != ".m3u8" {
		slog.Warn("wrong file extension", "extension", fileExt)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("file not found", "error", err)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			slog.Error("failed to stat file", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	if info.IsDir() {
		slog.Warn("directory access forbidden")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	if r.Method == http.MethodHead {
		setHeaders(w)
		if fileExt != ".m3u8" {
			w.Header().Set("Content-Type", h.ChunkMIME)
			w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
		} else {
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		}
		return
	}

	// get uid if possible (cookie set) and sid
	var (
		uid      string
		isNewUid bool
		sid      string = r.URL.Query().Get(h.SidName)
	)

	if sid == "" {
		sid = uuid.New().String()
	}

	uidCookie, err := r.Cookie(h.UidName)
	if err != nil && err != http.ErrNoCookie {
		slog.Error("failed to get uid cookie", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if uidCookie == nil {
		uid = r.URL.Query().Get(h.UidName)
		if uid == "" {
			uid = uuid.New().String()
			isNewUid = true
		}
		uidCookie = &http.Cookie{
			Name:     h.UidName,
			Value:    uid,
			Path:     "/",
			MaxAge:   31536000, // 1 year in seconds
			Secure:   true,
			HttpOnly: true,
		}
	} else {
		uid = uidCookie.Value
		isNewUid = false
	}
	http.SetCookie(w, uidCookie)

	file, err := os.Open(path)
	if err != nil {
		slog.Error("failed to open file", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	if fileExt != ".m3u8" {
		setHeaders(w)
		w.Header().Set("Content-Type", h.ChunkMIME)

		// Once io.Copy starts writing, the 200 status is already on the wire.
		// If the copy fails mid-stream, we can only log — the client will see
		// a truncated body.
		if _, err := io.Copy(w, file); err != nil {
			slog.Error("failed to copy chunk",
				"error", err,
				"path", path,
				"ip", r.RemoteAddr,
			)
			return
		}
		slog.Info("chunk",
			"path", path,
			"size", info.Size(),
			"ip", r.RemoteAddr,
			"user-agent", r.UserAgent(),
			"sid", sid,
			"uid", uid,
			"referer", r.Referer(),
		)
		if h.SessionTracker != nil {
			h.SessionTracker.Send(chunklog.ChunkEvent{
				Time:      time.Now(),
				Path:      path,
				ChunkSize: info.Size(),
				IP:        r.RemoteAddr,
				UserAgent: r.UserAgent(),
				Referer:   r.Referer(),
				SID:       sid,
				UID:       uid,
				Source:    chunklog.EventSourceHLS,
			})
		}
		return
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, h.BufferSize), h.BufferSize)
	outBuf := bytes.NewBuffer(make([]byte, 0, h.BufferSize))

	params := h.SidName + "=" + sid + "&" + h.UidName + "=" + uid
	for scanner.Scan() {
		var err error
		line := scanner.Text()
		switch {
		case line == "":
			// Preserve blank lines as-is.
			_, err = outBuf.WriteString("\n")
		case strings.HasPrefix(line, "#"):
			_, err = outBuf.WriteString(line + "\n")
		default:
			// Append session params; use '&' if the URI already has a query string.
			sep := "?"
			if strings.Contains(line, "?") {
				sep = "&"
			}
			_, err = outBuf.WriteString(line + sep + params + "\n")
		}
		if err != nil {
			slog.Error("failed to write output", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
	if err := scanner.Err(); err != nil {
		slog.Error("failed to scan file", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	setHeaders(w)
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	if _, err := outBuf.WriteTo(w); err != nil {
		slog.Error("failed to write output", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	slog.Info("playlist",
		"path", path,
		"ip", r.RemoteAddr,
		"user-agent", r.UserAgent(),
		"sid", sid,
		"uid", uid,
		"referer", r.Referer(),
	)
	if isNewUid {
		//TODO: store info in db
		slog.Info("new uid", "uid", uid)
	}
}

func setHeaders(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}
