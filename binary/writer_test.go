package binary

import (
	"bytes"
	"math"
	"testing"
)

func TestWriterPrimitives(t *testing.T) {
	w := NewWriter()

	w.Uint32(0x12345678)
	w.Int32(-1)
	w.Uint64(0x0102030405060708)
	w.Float32(1)
	w.Bool(true)
	w.Bool(false)
	w.Byte(0xab)

	want := []byte{
		0x78, 0x56, 0x34, 0x12,
		0xff, 0xff, 0xff, 0xff,
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01,
		0x00, 0x00, 0x80, 0x3f,
		0x01,
		0x00,
		0xab,
	}
	if got := w.Data(); !bytes.Equal(got, want) {
		t.Fatalf("data = % x, want % x", got, want)
	}
	if got := w.Len(); got != len(want) {
		t.Fatalf("len = %d, want %d", got, len(want))
	}
}

func TestWriterRoundTripsReaderPrimitives(t *testing.T) {
	w := NewWriter()

	w.Uint32(0x12345678)
	w.Int32(-12345)
	w.Uint64(0x0102030405060708)
	w.Float32(12.5)
	w.Bool(true)
	w.Bool(false)
	w.Byte(0xcd)

	r := NewReader(w.Data())
	if got := r.Uint32(); got != 0x12345678 {
		t.Fatalf("u32 = %#x, want 0x12345678", got)
	}
	if got := r.Int32(); got != -12345 {
		t.Fatalf("i32 = %d, want -12345", got)
	}
	if got := r.Uint64(); got != 0x0102030405060708 {
		t.Fatalf("u64 = %#x, want 0x0102030405060708", got)
	}
	if got := r.Float32(); got != 12.5 {
		t.Fatalf("f32 = %v, want 12.5", got)
	}
	if got := r.Bool(); !got {
		t.Fatal("first bool = false, want true")
	}
	if got := r.Bool(); got {
		t.Fatal("second bool = true, want false")
	}
	if got := r.Byte(); got != 0xcd {
		t.Fatalf("byte = %#x, want 0xcd", got)
	}
	if got := r.Remaining(); got != 0 {
		t.Fatalf("remaining = %d, want 0", got)
	}
}

func TestWriterString(t *testing.T) {
	w := NewWriter()

	w.String("hello")
	w.String("")
	w.String("ok")

	want := []byte{0x05, 'h', 'e', 'l', 'l', 'o', 0x00, 0x02, 'o', 'k'}
	if got := w.Data(); !bytes.Equal(got, want) {
		t.Fatalf("data = % x, want % x", got, want)
	}

	r := NewReader(w.Data())
	if got := r.String(); got != "hello" {
		t.Fatalf("first str = %q, want hello", got)
	}
	if got := r.String(); got != "" {
		t.Fatalf("second str = %q, want empty", got)
	}
	if got := r.String(); got != "ok" {
		t.Fatalf("third str = %q, want ok", got)
	}
}

func TestWriterString7BitLengthBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		length     int
		wantPrefix []byte
	}{
		{name: "zero", length: 0, wantPrefix: []byte{0x00}},
		{name: "one byte max", length: 127, wantPrefix: []byte{0x7f}},
		{name: "two byte min", length: 128, wantPrefix: []byte{0x80, 0x01}},
		{name: "two byte max", length: 16383, wantPrefix: []byte{0xff, 0x7f}},
		{name: "three byte min", length: 16384, wantPrefix: []byte{0x80, 0x80, 0x01}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWriter()
			w.String(string(bytesOf('x', tt.length)))

			got := w.Data()
			if !bytes.HasPrefix(got, tt.wantPrefix) {
				t.Fatalf("prefix = % x, want % x", got[:len(tt.wantPrefix)], tt.wantPrefix)
			}
			if len(got) != len(tt.wantPrefix)+tt.length {
				t.Fatalf("encoded length = %d, want %d", len(got), len(tt.wantPrefix)+tt.length)
			}

			r := NewReader(got)
			if got := len(r.String()); got != tt.length {
				t.Fatalf("round-trip string length = %d, want %d", got, tt.length)
			}
			if got := r.Remaining(); got != 0 {
				t.Fatalf("remaining = %d, want 0", got)
			}
		})
	}
}

func TestWriterBytesAppendsCopy(t *testing.T) {
	src := []byte{0x01, 0x02}
	w := NewWriter()

	w.Bytes(src)
	src[0] = 0xff

	if got, want := w.Data(), []byte{0x01, 0x02}; !bytes.Equal(got, want) {
		t.Fatalf("data = % x, want % x", got, want)
	}
}

func TestWriterDataReturnsCopy(t *testing.T) {
	w := NewWriter()
	w.Bytes([]byte{0x01, 0x02})

	got := w.Data()
	got[0] = 0xff

	if got, want := w.Data(), []byte{0x01, 0x02}; !bytes.Equal(got, want) {
		t.Fatalf("data after caller mutation = % x, want % x", got, want)
	}
}

func TestWriterFloatPreservesBits(t *testing.T) {
	w := NewWriter()
	nan := math.Float32frombits(0x7fc00000)

	w.Float32(nan)

	got := w.Data()
	want := []byte{0x00, 0x00, 0xc0, 0x7f}
	if !bytes.Equal(got, want) {
		t.Fatalf("data = % x, want % x", got, want)
	}
	if got := math.Float32bits(NewReader(got).Float32()); got != 0x7fc00000 {
		t.Fatalf("round-trip bits = %#x, want 0x7fc00000", got)
	}
}

func TestWriterRejectsNegative7BitEncodedInt(t *testing.T) {
	w := NewWriter()

	err := mustPanic(t, func() { w.write7BitEncodedInt(-1) })
	if err == nil || err.Error() != "fch: invalid negative 7-bit encoded integer -1" {
		t.Fatalf("panic = %v, want invalid negative 7-bit encoded integer", err)
	}
	if got := w.Len(); got != 0 {
		t.Fatalf("len = %d, want 0 after failed write", got)
	}
}

func TestWriterRejectsOversized7BitEncodedInt(t *testing.T) {
	w := NewWriter()

	err := mustPanic(t, func() { w.write7BitEncodedInt(max7BitEncodedInt + 1) })
	if err == nil || err.Error() != "fch: invalid oversized 7-bit encoded integer 2147483648" {
		t.Fatalf("panic = %v, want invalid oversized 7-bit encoded integer", err)
	}
	if got := w.Len(); got != 0 {
		t.Fatalf("len = %d, want 0 after failed write", got)
	}
}

func TestWriterMax7BitEncodedInt(t *testing.T) {
	w := NewWriter()

	w.write7BitEncodedInt(max7BitEncodedInt)

	want := []byte{0xff, 0xff, 0xff, 0xff, 0x07}
	if got := w.Data(); !bytes.Equal(got, want) {
		t.Fatalf("data = % x, want % x", got, want)
	}
}

func TestWriterOutputMatchesReaderFixture(t *testing.T) {
	w := NewWriter()

	w.Uint32(0x12345678)
	w.Int32(-1)
	w.Uint64(0x0102030405060708)
	w.Float32(1)
	w.Bool(true)
	w.Byte(0xab)

	want := []byte{
		0x78, 0x56, 0x34, 0x12,
		0xff, 0xff, 0xff, 0xff,
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01,
		0x00, 0x00, 0x80, 0x3f,
		0x01,
		0xab,
	}
	if !bytes.Equal(w.Data(), want) {
		t.Fatalf("Writer fixture = % x, want Reader fixture % x", w.Data(), want)
	}
}
