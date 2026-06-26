package fch

type Map struct {
	Offset           int    `json:"offset"`
	CompressedLength uint32 `json:"compressedLength"`
	StoredLength     uint32 `json:"storedLength"`
	Raw              []byte `json:"-"`
}
