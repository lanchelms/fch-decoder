package valheim

import (
	"encoding/binary"
	"fmt"
	wire "github.com/lanchelms/fch-decoder/binary"
)

type Map struct {
	Offset           int    `json:"offset"`
	CompressedLength uint32 `json:"compressedLength"`
	StoredLength     uint32 `json:"storedLength"`
	Raw              []byte `json:"-"`
}

func readMapSection(data []byte, startOffset int, payloadEnd int) (Map, int, error) {
	r := wire.NewReader(data).Slice(startOffset, payloadEnd)
	m := Map{Offset: startOffset}
	r.Bool()
	count := r.Uint32()
	if uint64(count)*60 > uint64(r.Remaining()) {
		return m, 0, fmt.Errorf("fch: invalid world count %d", count)
	}
	for range count {
		r.Bytes(59)
		if r.Bool() {
			n := r.Uint32()
			offset := startOffset + r.Position()
			blob := r.Bytes(int(n))
			if len(blob) >= 8 {
				m.StoredLength = n
				m.CompressedLength = binary.LittleEndian.Uint32(blob[4:8])
				m.Offset = offset + 8
			}
		}
	}
	end := startOffset + r.Position()
	m.Raw = append([]byte(nil), data[startOffset:end]...)
	return m, end, nil
}
