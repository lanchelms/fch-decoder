package binary

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type Reader struct {
	data []byte
	pos  int
}

func NewReader(data []byte) *Reader {
	return &Reader{data: data}
}

func (r *Reader) Data() []byte {
	return r.data
}

func (r *Reader) Position() int {
	return r.pos
}

func (r *Reader) SetPosition(pos int) {
	if pos < 0 || pos > len(r.data) {
		panic(io.ErrUnexpectedEOF)
	}
	r.pos = pos
}

func (r *Reader) Slice(start int, end int) *Reader {
	if start < 0 || end < start || end > len(r.data) {
		panic(io.ErrUnexpectedEOF)
	}
	return NewReader(r.data[start:end])
}

func (r *Reader) Remaining() int {
	if r.pos >= len(r.data) {
		return 0
	}
	return len(r.data) - r.pos
}

func (r *Reader) Capacity(count uint32) int {
	capacity := int(count)
	if capacity > r.Remaining() {
		return r.Remaining()
	}
	return capacity
}

func (r *Reader) need(n int) bool {
	return n >= 0 && r.pos+n <= len(r.data)
}

func (r *Reader) Bytes(n int) []byte {
	if !r.need(n) {
		panic(io.ErrUnexpectedEOF)
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b
}

func (r *Reader) Uint32() uint32 {
	return binary.LittleEndian.Uint32(r.Bytes(4))
}

func (r *Reader) Int32() int32 {
	return int32(r.Uint32())
}

func (r *Reader) Uint64() uint64 {
	return binary.LittleEndian.Uint64(r.Bytes(8))
}

func (r *Reader) Float32() float32 {
	return math.Float32frombits(r.Uint32())
}

func (r *Reader) Bool() bool {
	return r.Byte() != 0
}

func (r *Reader) Byte() byte {
	return r.Bytes(1)[0]
}

func (r *Reader) String() string {
	n := r.read7BitEncodedInt()
	b := r.Bytes(n)
	return string(b)
}

func (r *Reader) read7BitEncodedInt() int {
	var count uint32
	var shift uint
	for shift != 35 {
		b := r.Byte()
		count |= uint32(b&0x7f) << shift
		if b&0x80 == 0 {
			return int(count)
		}
		shift += 7
	}
	panic(fmt.Errorf("fch: invalid 7-bit encoded integer"))
}
