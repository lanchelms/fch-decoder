package binary

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

func (w *Writer) Bytes(b []byte) {
	w.buf = append(w.buf, b...)
}

func (w *Writer) Uint32(v uint32) {
	w.buf = binary.LittleEndian.AppendUint32(w.buf, v)
}

func (w *Writer) Int32(v int32) {
	w.Uint32(uint32(v))
}

func (w *Writer) Uint64(v uint64) {
	w.buf = binary.LittleEndian.AppendUint64(w.buf, v)
}

func (w *Writer) Float32(v float32) {
	w.Uint32(math.Float32bits(v))
}

func (w *Writer) Bool(v bool) {
	if v {
		w.Byte(1)
		return
	}
	w.Byte(0)
}

func (w *Writer) Byte(v byte) {
	w.buf = append(w.buf, v)
}

func (w *Writer) String(v string) {
	w.write7BitEncodedInt(len(v))
	w.Bytes([]byte(v))
}

func (w *Writer) write7BitEncodedInt(v int) {
	if v < 0 {
		panic(fmt.Errorf("fch: invalid negative 7-bit encoded integer %d", v))
	}
	if v > max7BitEncodedInt {
		panic(fmt.Errorf("fch: invalid oversized 7-bit encoded integer %d", v))
	}
	for v >= 0x80 {
		w.Byte(byte(v) | 0x80)
		v >>= 7
	}
	w.Byte(byte(v))
}
