package valheim

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Map struct {
	Offset           int    `json:"offset"`
	CompressedLength uint32 `json:"compressedLength"`
	StoredLength     uint32 `json:"storedLength"`
	Raw              []byte `json:"-"`
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
