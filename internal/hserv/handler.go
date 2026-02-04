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

	"github.com/google/uuid"
)

func (h *HServ) handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		slog.Error("method not allowed", "method", r.Method)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	path := filepath.Join(h.RootDir, filepath.FromSlash(r.URL.Path))
	rel, relErr := filepath.Rel(h.RootDir, path)
	if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		slog.Error("wrong path", "path", r.URL.Path)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	fileExt := filepath.Ext(path)
	if fileExt != h.ChunkExt && fileExt != ".m3u8" {
		slog.Error("wrong file extension", "extension", fileExt)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Error("file not found", "error", err)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			slog.Error("failed to stat file", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	if info.IsDir() {
		slog.Error("directory access forbidden")
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

	uidCookie, err := r.Cookie(h.UidName)
	if err != nil && err != http.ErrNoCookie {
		slog.Error("failed to get uid cookie", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if uidCookie == nil {
		uid = uuid.New().String()
		isNewUid = true
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
	w.Header().Set("Set-Cookie", uidCookie.String())

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

		status := http.StatusOK
		if _, err := io.Copy(w, file); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			slog.Error("failed to copy file", "error", err)
			status = http.StatusInternalServerError
		}
		// log only chunks
		slog.Info("chunk",
			"status", status,
			"method", r.Method,
			"path", path,
			"size", info.Size(),
			"ip", r.RemoteAddr,
			"user-agent", r.UserAgent(),
			"sid", sid,
		)
		return
	}

	if sid == "" {
		sid = uuid.New().String()
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, h.BufferSize), h.BufferSize)
	outBuf := bytes.NewBuffer(make([]byte, 0, h.BufferSize))

	for scanner.Scan() {
		var err error
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			_, err = outBuf.WriteString(line + "\n")
		} else {
			_, err = outBuf.WriteString(line + "?" + h.SidName + "=" + sid + "\n")
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
	if isNewUid {
		//TODO: store info in db
		slog.Info("new uid", "uid", uid)
	}
}

func setHeaders(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD")

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}
