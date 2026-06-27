package fch

import (
	"bytes"
	"fmt"
	"io"

	"github.com/lanchelms/fch-decoder/binary"
	"github.com/lanchelms/fch-decoder/valheim"
)

const (
	fileLengthSize = 4
	trailerSize    = 68
	fileOverhead   = fileLengthSize + trailerSize
)

func Decode(r io.Reader) (*valheim.Character, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return DecodeBytes(data)
}

func DecodeBytes(data []byte) (character *valheim.Character, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			character = nil
			err = fmt.Errorf("fch: decode failed: %v", recovered)
		}
	}()

	if len(data) < fileOverhead+16 {
		return nil, fmt.Errorf("fch: file too short: %d bytes", len(data))
	}

	rd := binary.NewReader(data)
	fileLength := rd.Uint32()
	payloadEnd := fileLengthSize + int(fileLength)
	if payloadEnd+trailerSize != len(data) {
		return nil, fmt.Errorf("fch: length header %d does not match file size %d", fileLength, len(data))
	}

	c := &valheim.Character{FileLength: fileLength}
	c.Decode(rd)
	c.Trailer.Offset = payloadEnd
	c.Trailer.Length = rd.Uint32()
	if c.Trailer.Length != payloadHashSize {
		return nil, fmt.Errorf("fch: unexpected trailer hash length %d", c.Trailer.Length)
	}
	c.Trailer.Hash = append([]byte(nil), rd.Bytes(payloadHashSize)...)
	c.Trailer.HashValid = bytes.Equal(hash(data[fileLengthSize:payloadEnd]), c.Trailer.Hash)
	return c, nil
}
