package fch

import (
	"fmt"
	"io"
)

const (
	fileLengthSize = 4
	trailerSize    = 68
	fileOverhead   = fileLengthSize + trailerSize
)

type decoder interface {
	Decode(*Reader)
}

func Decode(r io.Reader) (*Character, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return DecodeBytes(data)
}

func DecodeBytes(data []byte) (character *Character, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			character = nil
			err = fmt.Errorf("fch: decode failed: %v", recovered)
		}
	}()

	if len(data) < fileOverhead+16 {
		return nil, fmt.Errorf("fch: file too short: %d bytes", len(data))
	}

	rd := NewReader(data)
	c := &Character{}
	c.FileLength = rd.u32()
	c.Version = rd.u32()
	c.PlayerStatCount = rd.u32()
	if int(c.FileLength)+fileOverhead != len(data) {
		return nil, fmt.Errorf("fch: length header %d does not match file size %d", c.FileLength, len(data))
	}

	rd = NewReader(data)
	c.Decode(rd)
	return c, nil
}
