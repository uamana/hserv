package hserv

import (
	"net/http"
	"strconv"
	"time"

	"github.com/uamana/hserv/internal/chunklog"
)

func (h *HServ) icecastHandler(w http.ResponseWriter, r *http.Request) {
	mount := r.URL.Query().Get("mount")
	ip := r.URL.Query().Get("ip")
	userAgent := r.URL.Query().Get("agent")

	if mount == "" || ip == "" || userAgent == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	icecastID, err := strconv.ParseInt(r.URL.Query().Get("client"), 10, 64)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	// TODO: check if icecastID is valid
	if icecastID == 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if h.SessionTracker != nil {
		h.SessionTracker.Send(chunklog.ChunkEvent{
			Time:      time.Now(),
			IP:        ip,
			UserAgent: userAgent,
			IcecastID: icecastID,
			Mount:     mount,
			Source:    chunklog.EventSourceIceCast,
		})
	}
	w.Header().Set("icecast-auth-user", "1")
	w.WriteHeader(http.StatusNoContent)
}
