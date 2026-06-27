package valheim

import "github.com/lanchelms/fch-decoder/binary"

func NewReader(data []byte) *binary.Reader {
	return binary.NewReader(data)
}

func NewWriter() *binary.Writer {
	return binary.NewWriter()
}
