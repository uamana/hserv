package chunklog

import (
	"net"
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
}

type ChunkQuality byte

const (
	ChunkQualityLoFi ChunkQuality = iota
	ChunkQualityMidFi
	ChunkQualityHiFi
)

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

var CodecTypes = map[string]Codec{
	"aac":         CodecAAC,
	"mp3":         CodecMP3,
	"ac3":         CodecAC3,
	"eac3":        CodecEAC3,
	"dolby_atmos": CodecDolbyAtmos,
	"flac":        CodecFLAC,
	"opus":        CodecOpus,
	"speex":       CodecSpeex,
	"vorbis":      CodecVorbis,
	"":            CodecUnknown,
}

var CodecNames = map[Codec]string{
	CodecAAC:        "aac",
	CodecMP3:        "mp3",
	CodecAC3:        "ac3",
	CodecEAC3:       "eac3",
	CodecDolbyAtmos: "dolby_atmos",
	CodecFLAC:       "flac",
	CodecOpus:       "opus",
	CodecSpeex:      "speex",
	CodecVorbis:     "vorbis",
}

func (c Codec) String() string {
	name, ok := CodecNames[c]
	if !ok {
		return "unknown"
	}
	return name
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

func parseEvent(event *ChunkEvent, dbEvent *DBEvent) {
	if dbEvent == nil {
		return
	}

	// Basic fields copied directly from the event.
	dbEvent.Time = event.Time
	dbEvent.Path = event.Path
	dbEvent.IP = net.ParseIP(event.IP)
	dbEvent.Referer = event.Referer
	dbEvent.SID = event.SID
	dbEvent.UID = event.UID

	// User agent parsing and enrichment.
	if event.UserAgent == "" {
		return
	}

	ua := useragent.NewParser().Parse(event.UserAgent)

	dbEvent.UABrowser = ua.Browser().String()
	dbEvent.UABrowserVersion = ua.BrowserVersion()
	dbEvent.UADevice = ua.Device().String()

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
