package chunklog

import (
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	useragent "github.com/medama-io/go-useragent"
)

// ChunkEvent represents a chunk request event for analytics logging.
// Pass by value; struct is small.
type ChunkEvent struct {
	Time      time.Time
	Path      string
	IP        string
	UserAgent string
	Referer   string
	SID       uuid.UUID
	UID       uuid.UUID
	ChunkSize int64
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

var chunkRequestColumns = []string{
	"time",
	"path",
	"ip",
	"referer",
	"sid",
	"uid",
	"chunk_codec",
	"chunk_quality",
	"chunk_size",
	"chunk_duration",
	"chunk_timestamp",
	"chunk_sequence",
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

type DBEvent struct {
	Time               time.Time
	Path               string
	IP                 net.IP
	Referer            string
	SID                uuid.UUID
	UID                uuid.UUID
	ChunkCodec         Codec
	ChunkQuality       ChunkQuality
	ChunkSize          int64
	ChunkDuration      int
	ChunkTimestamp     time.Time
	ChunkSequence      int64
	UABrowser          string
	UABrowserVersion   string
	UADevice           string
	UAOS               string
	UAIsDesktop        bool
	UAIsMobile         bool
	UAIsTablet         bool
	UAIsTV             bool
	UAIsBot            bool
	UAIsAndroid        bool
	UAIsIOS            bool
	UAIsWindows        bool
	UAIsLinux          bool
	UAIsMac            bool
	UAIsOpenBSD        bool
	UAIsChromeOS       bool
	UAIsChrome         bool
	UAIsFirefox        bool
	UAIsSafari         bool
	UAIsEdge           bool
	UAIsOpera          bool
	UAIsSamsungBrowser bool
	UAIsVivaldi        bool
	UAIsYandexBrowser  bool
}

func parseEvent(event *ChunkEvent, dbEvent *DBEvent, parser *useragent.Parser) {
	if dbEvent == nil {
		return
	}

	// Basic fields copied directly from the event.
	dbEvent.Time = event.Time
	dbEvent.Path = event.Path

	if strings.Contains(event.IP, ":") {
		dbEvent.IP = net.ParseIP(strings.Split(event.IP, ":")[0])
	} else {
		dbEvent.IP = net.ParseIP(event.IP)
	}

	dbEvent.Referer = event.Referer
	dbEvent.SID = event.SID
	dbEvent.UID = event.UID

	// User agent parsing and enrichment.
	if event.UserAgent == "" {
		dbEvent.UABrowser = ""
		dbEvent.UABrowserVersion = ""
		dbEvent.UADevice = ""
		dbEvent.UAOS = ""

		dbEvent.UAIsDesktop = false
		dbEvent.UAIsMobile = false
		dbEvent.UAIsTablet = false
		dbEvent.UAIsTV = false
		dbEvent.UAIsBot = false

		dbEvent.UAIsAndroid = false
		dbEvent.UAIsIOS = false
		dbEvent.UAIsWindows = false
		dbEvent.UAIsLinux = false
		dbEvent.UAIsMac = false
		dbEvent.UAIsOpenBSD = false
		dbEvent.UAIsChromeOS = false

		dbEvent.UAIsChrome = false
		dbEvent.UAIsFirefox = false
		dbEvent.UAIsSafari = false
		dbEvent.UAIsEdge = false
		dbEvent.UAIsOpera = false
		dbEvent.UAIsSamsungBrowser = false
		dbEvent.UAIsVivaldi = false
		dbEvent.UAIsYandexBrowser = false
	} else {
		ua := parser.Parse(event.UserAgent)

		dbEvent.UABrowser = ua.Browser().String()
		dbEvent.UABrowserVersion = ua.BrowserVersion()
		dbEvent.UADevice = ua.Device().String()
		dbEvent.UAOS = ua.OS().String()

		dbEvent.UAIsDesktop = ua.IsDesktop()
		dbEvent.UAIsMobile = ua.IsMobile()
		dbEvent.UAIsTablet = ua.IsTablet()
		dbEvent.UAIsTV = ua.IsTV()
		dbEvent.UAIsBot = ua.IsBot()

		dbEvent.UAIsAndroid = ua.IsAndroidOS()
		dbEvent.UAIsIOS = ua.IsIOS()
		dbEvent.UAIsWindows = ua.IsWindows()
		dbEvent.UAIsLinux = ua.IsLinux()
		dbEvent.UAIsMac = ua.IsMacOS()
		dbEvent.UAIsOpenBSD = ua.IsOpenBSD()
		dbEvent.UAIsChromeOS = ua.IsChromeOS()

		dbEvent.UAIsChrome = ua.IsChrome()
		dbEvent.UAIsFirefox = ua.IsFirefox()
		dbEvent.UAIsSafari = ua.IsSafari()
		dbEvent.UAIsEdge = ua.IsEdge()
		dbEvent.UAIsOpera = ua.IsOpera()
		dbEvent.UAIsSamsungBrowser = ua.IsSamsungBrowser()
		dbEvent.UAIsVivaldi = ua.IsVivaldi()
		dbEvent.UAIsYandexBrowser = ua.IsYandexBrowser()
	}

	dbEvent.ChunkSize = event.ChunkSize
	if event.Path != "" {
		chunkFileName := filepath.Base(event.Path)
		parts := strings.Split(chunkFileName, "_")
		if len(parts) != 5 {
			return
		}
		dbEvent.ChunkCodec = CodecFromString(parts[0])
		dbEvent.ChunkQuality = ChunkQualityFromString(parts[1])
		chunkTimestamp, _ := strconv.ParseInt(parts[2], 10, 64)
		dbEvent.ChunkTimestamp = time.Unix(chunkTimestamp, 0)
		chunkDuration, _ := strconv.ParseFloat(parts[3], 64)
		dbEvent.ChunkDuration = int(chunkDuration*100) * 10
		dbEvent.ChunkSequence, _ = strconv.ParseInt(parts[3], 10, 64)
	} else {
		dbEvent.ChunkCodec = CodecUnknown
		dbEvent.ChunkQuality = ChunkQualityUnknown
		dbEvent.ChunkTimestamp = time.Time{}
		dbEvent.ChunkSequence = 0
		dbEvent.ChunkDuration = 0
	}
}
