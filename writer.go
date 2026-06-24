package fch

import (
	"encoding/binary"
	"fmt"
	"math"
)

const max7BitEncodedInt = int(1<<31 - 1)

type writer struct {
	buf []byte
}

func newWriter() *writer {
	return &writer{}
}

func (w *writer) data() []byte {
	return append([]byte(nil), w.buf...)
}

func (w *writer) len() int {
	return len(w.buf)
}

func (w *writer) bytes(b []byte) {
	w.buf = append(w.buf, b...)
}

func (w *writer) u32(v uint32) {
	w.buf = binary.LittleEndian.AppendUint32(w.buf, v)
}

func (w *writer) i32(v int32) {
	w.u32(uint32(v))
}

func (w *writer) u64(v uint64) {
	w.buf = binary.LittleEndian.AppendUint64(w.buf, v)
}

func (w *writer) f32(v float32) {
	w.u32(math.Float32bits(v))
}

func (w *writer) bool(v bool) {
	if v {
		w.byte(1)
		return
	}
	w.byte(0)
}

func (w *writer) byte(v byte) {
	w.buf = append(w.buf, v)
}

func (w *writer) str(v string) {
	w.write7BitEncodedInt(len(v))
	w.bytes([]byte(v))
}

func (w *writer) vector3(v Vector3) {
	w.f32(v.X)
	w.f32(v.Y)
	w.f32(v.Z)
}

func (w *writer) write7BitEncodedInt(v int) {
	if v < 0 {
		panic(fmt.Errorf("fch: invalid negative 7-bit encoded integer %d", v))
	}
	if v > max7BitEncodedInt {
		panic(fmt.Errorf("fch: invalid oversized 7-bit encoded integer %d", v))
	}
	for v >= 0x80 {
		w.byte(byte(v) | 0x80)
		v >>= 7
	}
	w.byte(byte(v))
}
