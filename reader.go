package fch

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type reader struct {
	data []byte
	pos  int
	err  error
}

func newReader(data []byte) *reader {
	return &reader{data: data}
}

func (r *reader) remaining() int {
	if r.pos >= len(r.data) {
		return 0
	}
	return len(r.data) - r.pos
}

func (r *reader) need(n int) bool {
	if r.err != nil {
		return false
	}
	if n < 0 || r.pos+n > len(r.data) {
		r.err = io.ErrUnexpectedEOF
		return false
	}
	return true
}

func (r *reader) bytes(n int) []byte {
	if !r.need(n) {
		return nil
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b
}

func (r *reader) u32() uint32 {
	b := r.bytes(4)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}

func (r *reader) i32() int32 {
	return int32(r.u32())
}

func (r *reader) u64() uint64 {
	b := r.bytes(8)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint64(b)
}

func (r *reader) f32() float32 {
	return math.Float32frombits(r.u32())
}

func (r *reader) bool() bool {
	return r.byte() != 0
}

func (r *reader) byte() byte {
	b := r.bytes(1)
	if len(b) != 1 {
		return 0
	}
	return b[0]
}

func (r *reader) str() string {
	n := r.read7BitEncodedInt()
	if r.err != nil {
		return ""
	}
	b := r.bytes(n)
	return string(b)
}

func (r *reader) stringList() []string {
	count := r.u32()
	out := make([]string, 0, count)
	for i := uint32(0); i < count; i++ {
		out = append(out, r.str())
	}
	return out
}

func (r *reader) read7BitEncodedInt() int {
	var count uint32
	var shift uint
	for shift != 35 {
		b := r.bytes(1)
		if b == nil {
			return 0
		}
		count |= uint32(b[0]&0x7f) << shift
		if b[0]&0x80 == 0 {
			return int(count)
		}
		shift += 7
	}
	r.err = fmt.Errorf("fch: invalid 7-bit encoded integer")
	return 0
}
