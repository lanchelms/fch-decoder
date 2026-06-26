package fch

import (
	"encoding/binary"
	"fmt"
	"math"
)

const max7BitEncodedInt = int(1<<31 - 1)

type Writer struct {
	buf []byte
}

func NewWriter() *Writer {
	return &Writer{}
}

func (w *Writer) Data() []byte {
	return append([]byte(nil), w.buf...)
}

func (w *Writer) Len() int {
	return len(w.buf)
}

func (w *Writer) bytes(b []byte) {
	w.buf = append(w.buf, b...)
}

func (w *Writer) u32(v uint32) {
	w.buf = binary.LittleEndian.AppendUint32(w.buf, v)
}

func (w *Writer) i32(v int32) {
	w.u32(uint32(v))
}

func (w *Writer) u64(v uint64) {
	w.buf = binary.LittleEndian.AppendUint64(w.buf, v)
}

func (w *Writer) f32(v float32) {
	w.u32(math.Float32bits(v))
}

func (w *Writer) bool(v bool) {
	if v {
		w.byte(1)
		return
	}
	w.byte(0)
}

func (w *Writer) byte(v byte) {
	w.buf = append(w.buf, v)
}

func (w *Writer) str(v string) {
	w.write7BitEncodedInt(len(v))
	w.bytes([]byte(v))
}

func (w *Writer) write7BitEncodedInt(v int) {
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
