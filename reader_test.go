package fch

import (
	"errors"
	"io"
	"math"
	"testing"
)

func TestReaderPrimitives(t *testing.T) {
	r := NewReader([]byte{
		0x78, 0x56, 0x34, 0x12,
		0xff, 0xff, 0xff, 0xff,
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01,
		0x00, 0x00, 0x80, 0x3f,
		0x01,
		0xab,
	})

	if got := r.u32(); got != 0x12345678 {
		t.Fatalf("u32 = %#x, want 0x12345678", got)
	}
	if got := r.i32(); got != -1 {
		t.Fatalf("i32 = %d, want -1", got)
	}
	if got := r.u64(); got != 0x0102030405060708 {
		t.Fatalf("u64 = %#x, want 0x0102030405060708", got)
	}
	if got := r.f32(); got != 1 {
		t.Fatalf("f32 = %v, want 1", got)
	}
	if got := r.bool(); !got {
		t.Fatal("bool = false, want true")
	}
	if got := r.byte(); got != 0xab {
		t.Fatalf("byte = %#x, want 0xab", got)
	}
	if got := r.remaining(); got != 0 {
		t.Fatalf("remaining = %d, want 0", got)
	}
}

func TestReaderString(t *testing.T) {
	r := NewReader([]byte{0x05, 'h', 'e', 'l', 'l', 'o', 0x02, 'o', 'k'})

	if got := r.str(); got != "hello" {
		t.Fatalf("first str = %q, want hello", got)
	}
	if got := r.str(); got != "ok" {
		t.Fatalf("second str = %q, want ok", got)
	}
	if got := r.remaining(); got != 0 {
		t.Fatalf("remaining = %d, want 0", got)
	}
}

func TestReaderStringLong7BitLength(t *testing.T) {
	r := NewReader(append([]byte{0x82, 0x01}, bytesOf('x', 130)...))

	got := r.str()
	if len(got) != 130 {
		t.Fatalf("len(str) = %d, want 130", len(got))
	}
	if got[0] != 'x' || got[129] != 'x' {
		t.Fatalf("str endpoints = %q/%q, want x/x", got[0], got[129])
	}
}

func TestReaderUnexpectedEOF(t *testing.T) {
	r := NewReader([]byte{0x01, 0x02, 0x03})

	err := mustPanic(t, func() { r.u32() })
	if got := r.remaining(); got != 3 {
		t.Fatalf("remaining = %d, want 3 after failed read", got)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("panic = %v, want unexpected EOF", err)
	}
}

func TestReaderInvalid7BitEncodedInt(t *testing.T) {
	r := NewReader([]byte{0x80, 0x80, 0x80, 0x80, 0x80})

	err := mustPanic(t, func() { r.str() })
	if err == nil || err.Error() != "fch: invalid 7-bit encoded integer" {
		t.Fatalf("panic = %v, want invalid 7-bit encoded integer", err)
	}
}

func TestReaderNeedRejectsNegativeLength(t *testing.T) {
	r := NewReader([]byte{0x01})

	if r.need(-1) {
		t.Fatal("need(-1) = true, want false")
	}
}

func TestReaderFloatPreservesBits(t *testing.T) {
	r := NewReader([]byte{0x00, 0x00, 0xc0, 0x7f})

	if got := r.f32(); !math.IsNaN(float64(got)) {
		t.Fatalf("f32 = %v, want NaN", got)
	}
}

func bytesOf(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

func mustPanic(t *testing.T, fn func()) (err error) {
	t.Helper()
	defer func() {
		v := recover()
		if v == nil {
			t.Fatal("function did not panic")
		}
		var ok bool
		err, ok = v.(error)
		if !ok {
			t.Fatalf("panic = %#v, want error", v)
		}
	}()
	fn()
	return nil
}
