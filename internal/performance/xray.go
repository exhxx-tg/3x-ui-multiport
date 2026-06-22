package performance

import (
	"bytes"
	"sync"
)

var xrayConfigBufferPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func GetXrayConfigBuffer() *bytes.Buffer {
	buf := xrayConfigBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func PutXrayConfigBuffer(buf *bytes.Buffer) {
	buf.Reset()
	xrayConfigBufferPool.Put(buf)
}

var xrayTrafficBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 0, 4096)
		return &buf
	},
}

func GetTrafficBuffer() *[]byte {
	return xrayTrafficBufferPool.Get().(*[]byte)
}

func PutTrafficBuffer(buf *[]byte) {
	*buf = (*buf)[:0]
	xrayTrafficBufferPool.Put(buf)
}

type XrayConnPool struct {
	address string
	conns   chan struct{}
}

func NewXrayConnPool(address string, maxConns int) *XrayConnPool {
	if maxConns <= 0 {
		maxConns = 4
	}
	return &XrayConnPool{
		address: address,
		conns:   make(chan struct{}, maxConns),
	}
}

func (p *XrayConnPool) Acquire() {
	p.conns <- struct{}{}
}

func (p *XrayConnPool) Release() {
	<-p.conns
}

type TrafficBatch struct {
	mu      sync.Mutex
	entries map[string]*TrafficEntry
}

type TrafficEntry struct {
	Up   int64
	Down int64
}

func NewTrafficBatch() *TrafficBatch {
	return &TrafficBatch{
		entries: make(map[string]*TrafficEntry),
	}
}

func (tb *TrafficBatch) Add(key string, up, down int64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	entry, ok := tb.entries[key]
	if !ok {
		entry = &TrafficEntry{}
		tb.entries[key] = entry
	}
	entry.Up += up
	entry.Down += down
}

func (tb *TrafficBatch) Flush() map[string]*TrafficEntry {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	result := tb.entries
	tb.entries = make(map[string]*TrafficEntry)
	return result
}

func (tb *TrafficBatch) Len() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return len(tb.entries)
}
