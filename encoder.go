package fch

import (
	"fmt"
	"io"
	"math"

	"github.com/lanchelms/fch-decoder/binary"
	"github.com/lanchelms/fch-decoder/valheim"
)

func Encode(w io.Writer, c *valheim.Character) error {
	data, err := EncodeBytes(c)
	if err != nil {
		return err
	}
	n, err := w.Write(data)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}

func EncodeBytes(c *valheim.Character) ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	payload := binary.NewWriter()
	c.Encode(payload)
	payloadBytes := payload.Data()
	if len(payloadBytes) > math.MaxUint32 {
		return nil, fmt.Errorf("fch: payload too large: %d bytes", len(payloadBytes))
	}

	w := binary.NewWriter()
	w.Uint32(uint32(len(payloadBytes)))
	w.Bytes(payloadBytes)
	w.Uint32(payloadHashSize)
	w.Bytes(hash(payloadBytes))
	return w.Data(), nil
}
