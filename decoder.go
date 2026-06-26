package fch

import (
	"bytes"
	"encoding/binary"
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

func currentPayloadHash(data []byte, payloadLen uint32) []byte {
	return payloadHash(data[fileLengthSize : fileLengthSize+int(payloadLen)])
}

func readMapSection(data []byte, startOffset int, payloadEnd int) (Map, int, error) {
	firstSpawn, worldCount, ok := readMapPrefix(data, startOffset, payloadEnd)
	if ok && firstSpawn <= 1 && worldCount == 0 {
		return Map{
			Offset: startOffset,
			Raw:    append([]byte(nil), data[startOffset:startOffset+5]...),
		}, startOffset + 5, nil
	}

	gzipOffset := bytes.Index(data[startOffset:], []byte{0x1f, 0x8b, 0x08})
	if gzipOffset < 0 {
		return Map{}, 0, fmt.Errorf("fch: gzip map block not found")
	}
	gzipOffset += startOffset
	if gzipOffset < 12 {
		return Map{}, 0, fmt.Errorf("fch: gzip map block starts too early")
	}

	storedLen := binary.LittleEndian.Uint32(data[gzipOffset-12 : gzipOffset-8])
	compressedLen := binary.LittleEndian.Uint32(data[gzipOffset-4 : gzipOffset])
	if gzipOffset+int(compressedLen) > payloadEnd {
		return Map{}, 0, fmt.Errorf("fch: invalid compressed map length %d at offset %d", compressedLen, gzipOffset)
	}

	return Map{
		Offset:           gzipOffset,
		CompressedLength: compressedLen,
		StoredLength:     storedLen,
		Raw:              append([]byte(nil), data[startOffset:gzipOffset+int(compressedLen)]...),
	}, gzipOffset + int(compressedLen), nil
}

func readMapPrefix(data []byte, startOffset int, payloadEnd int) (byte, uint32, bool) {
	if startOffset+5 > payloadEnd {
		return 0, 0, false
	}
	firstSpawn := data[startOffset]
	worldCount := binary.LittleEndian.Uint32(data[startOffset+1 : startOffset+5])
	return firstSpawn, worldCount, true
}
