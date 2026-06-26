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

func (r *reader) capacity(count uint32) int {
	capacity := int(count)
	if capacity > r.remaining() {
		return r.remaining()
	}
	return capacity
}

func (r *reader) need(n int) bool {
	return n >= 0 && r.pos+n <= len(r.data)
}

func (r *reader) bytes(n int) []byte {
	if !r.need(n) {
		panic(io.ErrUnexpectedEOF)
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b
}

func (r *reader) u32() uint32 {
	return binary.LittleEndian.Uint32(r.bytes(4))
}

func (r *reader) i32() int32 {
	return int32(r.u32())
}

func (r *reader) u64() uint64 {
	return binary.LittleEndian.Uint64(r.bytes(8))
}

func (r *reader) f32() float32 {
	return math.Float32frombits(r.u32())
}

func (r *reader) bool() bool {
	return r.byte() != 0
}

func (r *reader) byte() byte {
	return r.bytes(1)[0]
}

func (r *reader) str() string {
	n := r.read7BitEncodedInt()
	b := r.bytes(n)
	return string(b)
}

func (r *reader) vector3() Vector3 {
	return Vector3{X: r.f32(), Y: r.f32(), Z: r.f32()}
}

func (r *reader) read7BitEncodedInt() int {
	var count uint32
	var shift uint
	for shift != 35 {
		b := r.byte()
		count |= uint32(b&0x7f) << shift
		if b&0x80 == 0 {
			return int(count)
		}
		shift += 7
	}
	panic(fmt.Errorf("fch: invalid 7-bit encoded integer"))
}
