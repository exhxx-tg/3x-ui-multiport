package performance

import (
	"bytes"
	"sync"
)

var (
	bytesBufferPool = sync.Pool{
		New: func() any { return new(bytes.Buffer) },
	}

	jsonBufferPool = sync.Pool{
		New: func() any { return make([]byte, 0, 4096) },
	}

	smallIntSlicePool = sync.Pool{
		New: func() any { return make([]int, 0, 64) },
	}

	stringSlicePool = sync.Pool{
		New: func() any { return make([]string, 0, 32) },
	}

	mapStringAnyPool = sync.Pool{
		New: func() any { return make(map[string]any, 16) },
	}

	mapStringStringPool = sync.Pool{
		New: func() any { return make(map[string]string, 8) },
	}
)

func GetBuffer() *bytes.Buffer {
	buf := bytesBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func PutBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bytesBufferPool.Put(buf)
}

func GetJSONBuffer() []byte {
	return jsonBufferPool.Get().([]byte)[:0]
}

func PutJSONBuffer(buf []byte) {
	buf = buf[:0]
	jsonBufferPool.Put(buf)
}

func GetSmallIntSlice() []int {
	s := smallIntSlicePool.Get().([]int)
	return s[:0]
}

func PutSmallIntSlice(s []int) {
	s = s[:0]
	smallIntSlicePool.Put(s)
}

func GetStringSlice() []string {
	s := stringSlicePool.Get().([]string)
	return s[:0]
}

func PutStringSlice(s []string) {
	s = s[:0]
	stringSlicePool.Put(s)
}

func GetMapStringAny() map[string]any {
	m := mapStringAnyPool.Get().(map[string]any)
	for k := range m {
		delete(m, k)
	}
	return m
}

func PutMapStringAny(m map[string]any) {
	for k := range m {
		delete(m, k)
	}
	mapStringAnyPool.Put(m)
}

func GetMapStringString() map[string]string {
	m := mapStringStringPool.Get().(map[string]string)
	for k := range m {
		delete(m, k)
	}
	return m
}

func PutMapStringString(m map[string]string) {
	for k := range m {
		delete(m, k)
	}
	mapStringStringPool.Put(m)
}
