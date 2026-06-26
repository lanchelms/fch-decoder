package fch

type Trailer struct {
	Offset    int    `json:"offset"`
	Length    uint32 `json:"length"`
	Hash      []byte `json:"hash"`
	HashValid bool   `json:"hashValid"`
}
