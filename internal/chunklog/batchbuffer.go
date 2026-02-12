package chunklog

import "errors"

var ErrIndexOutOfBounds = errors.New("index out of bounds")

type BatchBuffer struct {
	wIdx int
	rIdx int
	buf  []DBEvent
	err  error
}

func NewBatchBuffer(size int) *BatchBuffer {
	return &BatchBuffer{
		wIdx: 0,
		rIdx: -1,
		buf:  make([]DBEvent, size),
	}
}

func (b *BatchBuffer) Next() bool {
	b.rIdx++
	return b.rIdx < b.wIdx
}

func (b *BatchBuffer) Values() ([]interface{}, error) {
	if b.rIdx < 0 || b.rIdx >= len(b.buf) {
		b.err = ErrIndexOutOfBounds
		return nil, b.err
	}
	c := &b.buf[b.rIdx]
	return []interface{}{
		c.Time,
		c.Path,
		c.IP,
		c.Referer,
		c.SID,
		c.UID,
		c.ChunkCodec,
		c.ChunkQuality,
		c.ChunkSize,
		c.ChunkDuration,
		c.ChunkTimestamp,
		c.ChunkSequence,
		c.UABrowser,
		c.UABrowserVersion,
		c.UADevice,
		c.UAOS,
		c.UAIsDesktop,
		c.UAIsMobile,
		c.UAIsTablet,
		c.UAIsTV,
		c.UAIsBot,
		c.UAIsAndroid,
		c.UAIsIOS,
		c.UAIsWindows,
		c.UAIsLinux,
		c.UAIsMac,
		c.UAIsOpenBSD,
		c.UAIsChromeOS,
		c.UAIsChrome,
		c.UAIsFirefox,
		c.UAIsSafari,
		c.UAIsEdge,
		c.UAIsOpera,
		c.UAIsSamsungBrowser,
		c.UAIsVivaldi,
		c.UAIsYandexBrowser,
	}, nil
}

func (b *BatchBuffer) Err() error {
	return b.err
}

func (b *BatchBuffer) Reset() {
	b.rIdx = -1
	b.wIdx = 0
	b.err = nil
}

func (b *BatchBuffer) Add(event ChunkEvent) {
	if b.wIdx < 0 || b.wIdx >= len(b.buf) {
		b.err = ErrIndexOutOfBounds
		return
	}
	parseEvent(&event, &b.buf[b.wIdx])
	b.wIdx++
}

func (b *BatchBuffer) Len() int {
	return b.wIdx
}

func (b *BatchBuffer) IsFull() bool {
	return b.wIdx >= len(b.buf)
}
