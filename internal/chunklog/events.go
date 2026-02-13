package chunklog

import (
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	useragent "github.com/medama-io/go-useragent"
)

type EventSource byte

const (
	EventSourceHLS EventSource = iota
	EventSourceIceCast
	EventSourceUnknown EventSource = 255
)

var EventSourceNames = []string{"hls", "icecast"}

func (e EventSource) String() string {
	if int(e) < len(EventSourceNames) {
		return EventSourceNames[e]
	}
	return "unknown"
}

func EventSourceFromString(s string) EventSource {
	switch s {
	case "hls":
		return EventSourceHLS
	case "icecast":
		return EventSourceIceCast
	default:
		return EventSourceUnknown
	}
}

// ChunkEvent represents a chunk request event for analytics logging.
// Pass by value; struct is small.
type ChunkEvent struct {
	Time      time.Time
	Path      string
	IP        string
	UserAgent string
	Referer   string
	SID       string
	UID       string
	ChunkSize int64
	Source    EventSource
}

type ChunkQuality byte

const (
	ChunkQualityLoFi ChunkQuality = iota
	ChunkQualityMidFi
	ChunkQualityHiFi
	ChunkQualityUnknown ChunkQuality = 255
)

var ChunkQualityNames = []string{"lofi", "midfi", "hifi"}

func (c ChunkQuality) String() string {
	if int(c) < len(ChunkQualityNames) {
		return ChunkQualityNames[c]
	}
	return "unknown"
}

func ChunkQualityFromString(s string) ChunkQuality {
	switch s {
	case "lofi":
		return ChunkQualityLoFi
	case "midfi":
		return ChunkQualityMidFi
	case "hifi":
		return ChunkQualityHiFi
	default:
		return ChunkQualityUnknown
	}
}

type Codec byte

const (
	CodecAAC Codec = iota
	CodecMP3
	CodecAC3
	CodecEAC3
	CodecDolbyAtmos
	CodecFLAC
	CodecOpus
	CodecSpeex
	CodecVorbis
	CodecUnknown Codec = 255
)

var CodecNames = []string{"aac", "mp3", "ac3", "eac3", "dolby_atmos", "flac", "opus", "speex", "vorbis"}

func (c Codec) String() string {
	if int(c) < len(CodecNames) {
		return CodecNames[c]
	}
	return "unknown"
}

func CodecFromString(s string) Codec {
	switch s {
	case "aac":
		return CodecAAC
	case "mp3":
		return CodecMP3
	case "ac3":
		return CodecAC3
	case "eac3":
		return CodecEAC3
	case "dolby_atmos":
		return CodecDolbyAtmos
	case "flac":
		return CodecFLAC
	case "opus":
		return CodecOpus
	case "speex":
		return CodecSpeex
	case "vorbis":
		return CodecVorbis
	default:
		return CodecUnknown
	}
}

// TODO: add mount, for icecast use mount, for HLS use stream name (last dir in path)
// sessionColumns defines the column order for COPY INTO sessions.
var sessionColumns = []string{
	"sid",
	"uid",
	"source",
	"start_time",
	"end_time",
	"total_bytes",
	"codec",
	"quality",
	"ip",
	"referer",
	"ua_browser",
	"ua_browser_version",
	"ua_device",
	"ua_os",
	"ua_is_desktop",
	"ua_is_mobile",
	"ua_is_tablet",
	"ua_is_tv",
	"ua_is_bot",
	"ua_is_android",
	"ua_is_ios",
	"ua_is_windows",
	"ua_is_linux",
	"ua_is_mac",
	"ua_is_openbsd",
	"ua_is_chromeos",
	"ua_is_chrome",
	"ua_is_firefox",
	"ua_is_safari",
	"ua_is_edge",
	"ua_is_opera",
	"ua_is_samsung_browser",
	"ua_is_vivaldi",
	"ua_is_yandex_browser",
}

// Session represents an active or completed listening session.
type Session struct {
	SID              uuid.UUID
	UID              uuid.UUID
	Source           EventSource
	StartTime        time.Time
	LastActive       time.Time // written as end_time when flushed to DB
	TotalBytes       int64
	Codec            Codec
	Quality          ChunkQuality
	IP               net.IP
	Referer          string
	UABrowser        string
	UABrowserVersion string
	UADevice         string
	UAOS             string
}

// row returns the session as a row of values matching sessionColumns order.
func (s *Session) row() []interface{} {
	return []interface{}{
		s.SID, s.UID, s.Source, s.StartTime, s.LastActive, s.TotalBytes,
		s.Codec, s.Quality, s.IP, s.Referer,
		s.UABrowser, s.UABrowserVersion, s.UADevice, s.UAOS,
	}
}

// newSessionFromEvent creates a new Session from the first ChunkEvent for a SID.
func newSessionFromEvent(event *ChunkEvent, parser *useragent.Parser) *Session {
	s := &Session{
		StartTime:  event.Time,
		LastActive: event.Time,
		TotalBytes: event.ChunkSize,
		Source:     event.Source,
	}

	// TODO: maybe add chunk duration to LastActive time

	// Parse SID.
	sid, err := uuid.Parse(event.SID)
	if err != nil {
		s.SID = uuid.Nil
	} else {
		s.SID = sid
	}

	// Parse UID.
	uid, err := uuid.Parse(event.UID)
	if err != nil {
		s.UID = uuid.Nil
	} else {
		s.UID = uid
	}

	// Parse IP (strip port if present).
	if strings.Contains(event.IP, ":") {
		s.IP = net.ParseIP(strings.Split(event.IP, ":")[0])
	} else {
		s.IP = net.ParseIP(event.IP)
	}

	s.Referer = event.Referer

	// Parse User-Agent.
	if event.UserAgent != "" {
		ua := parser.Parse(event.UserAgent)
		s.UABrowser = ua.Browser().String()
		s.UABrowserVersion = ua.BrowserVersion()
		s.UADevice = ua.Device().String()
		s.UAOS = ua.OS().String()
	}

	// Parse codec and quality from chunk filename.
	if event.Path != "" {
		chunkFileName := filepath.Base(event.Path)
		parts := strings.Split(chunkFileName, "_")
		if len(parts) >= 2 {
			s.Codec = CodecFromString(parts[0])
			s.Quality = ChunkQualityFromString(parts[1])
		} else {
			s.Codec = CodecUnknown
			s.Quality = ChunkQualityUnknown
		}
	} else {
		s.Codec = CodecUnknown
		s.Quality = ChunkQualityUnknown
	}

	return s
}
