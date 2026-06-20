package fch

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

const trailerSize = 72

type Character struct {
	FileLength        uint32     `json:"fileLength"`
	Version           uint32     `json:"version"`
	PlayerDataVersion uint32     `json:"playerDataVersion"`
	SkillValues       []float32  `json:"skillValues,omitempty"`
	Map               MapSection `json:"map"`
	Player            PlayerData `json:"player"`
	Trailer           Trailer    `json:"trailer"`
	RemainingBytes    int        `json:"remainingBytes"`
}

type MapSection struct {
	Offset             int    `json:"offset"`
	CompressedLength   uint32 `json:"compressedLength"`
	StoredLength       uint32 `json:"storedLength"`
	UncompressedLength int    `json:"uncompressedLength"`
}

type Trailer struct {
	Offset  int    `json:"offset"`
	Unknown uint32 `json:"unknown"`
	Length  uint32 `json:"length"`
	Hash    []byte `json:"hash"`
}

type PlayerData struct {
	Name          string        `json:"name"`
	PlayerID      uint64        `json:"playerId"`
	UnknownA      uint32        `json:"unknownA"`
	UnknownB      uint32        `json:"unknownB"`
	UnknownFlags  []bool        `json:"unknownFlags"`
	Worlds        []WorldData   `json:"worlds"`
	KnownTexts    []StatEntry   `json:"knownTexts,omitempty"`
	EnemyStats    []StatEntry   `json:"enemyStats,omitempty"`
	MaterialStats []StatEntry   `json:"materialStats,omitempty"`
	RecipeStats   []StatEntry   `json:"recipeStats,omitempty"`
	GuardianPower GuardianPower `json:"guardianPower"`
	Inventory     []Item        `json:"inventory,omitempty"`
}

type WorldData struct {
	Name    string      `json:"name"`
	Time    float32     `json:"time"`
	Entries []StatEntry `json:"entries,omitempty"`
}

type StatEntry struct {
	Name  string  `json:"name"`
	Value float32 `json:"value"`
}

type GuardianPower struct {
	UnknownBool   bool      `json:"unknownBool"`
	UnknownIntA   uint32    `json:"unknownIntA"`
	UnknownIntB   uint32    `json:"unknownIntB"`
	UnknownFloats []float32 `json:"unknownFloats"`
	Name          string    `json:"name"`
	Cooldown      float32   `json:"cooldown"`
	UnknownIntC   uint32    `json:"unknownIntC"`
}

type Item struct {
	Name        string  `json:"name"`
	Stack       int32   `json:"stack"`
	Durability  float32 `json:"durability"`
	GridX       int32   `json:"gridX"`
	GridY       int32   `json:"gridY"`
	Equipped    bool    `json:"equipped"`
	Quality     int32   `json:"quality"`
	Variant     int32   `json:"variant"`
	CrafterID   uint64  `json:"crafterId"`
	CrafterName string  `json:"crafterName"`
	UnknownTail uint64  `json:"unknownTail"`
	UnknownByte byte    `json:"unknownByte"`
}

func Decode(r io.Reader) (*Character, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return DecodeBytes(data)
}

func DecodeBytes(data []byte) (*Character, error) {
	if len(data) < trailerSize+16 {
		return nil, fmt.Errorf("fch: file too short: %d bytes", len(data))
	}

	rd := newReader(data)
	c := &Character{}
	c.FileLength = rd.u32()
	c.Version = rd.u32()
	c.PlayerDataVersion = rd.u32()
	if rd.err != nil {
		return nil, rd.err
	}
	if int(c.FileLength)+trailerSize != len(data) {
		return nil, fmt.Errorf("fch: length header %d does not match file size %d", c.FileLength, len(data))
	}

	// Current samples store PlayerDataVersion float values before the map block.
	c.SkillValues = make([]float32, int(c.PlayerDataVersion))
	for i := range c.SkillValues {
		c.SkillValues[i] = rd.f32()
	}

	gzipOffset := bytes.Index(data[rd.pos:], []byte{0x1f, 0x8b, 0x08})
	if gzipOffset < 0 {
		return nil, fmt.Errorf("fch: gzip map block not found")
	}
	gzipOffset += rd.pos
	if gzipOffset < 12 {
		return nil, fmt.Errorf("fch: gzip map block starts too early")
	}
	storedLen := binary.LittleEndian.Uint32(data[gzipOffset-12 : gzipOffset-8])
	compressedLen := binary.LittleEndian.Uint32(data[gzipOffset-4 : gzipOffset])
	if int(compressedLen) < 0 || gzipOffset+int(compressedLen) > len(data)-trailerSize {
		return nil, fmt.Errorf("fch: invalid compressed map length %d at offset %d", compressedLen, gzipOffset)
	}
	uncompressed, err := gunzip(data[gzipOffset : gzipOffset+int(compressedLen)])
	if err != nil {
		return nil, fmt.Errorf("fch: decompress map block: %w", err)
	}
	c.Map = MapSection{
		Offset:             gzipOffset,
		CompressedLength:   compressedLen,
		StoredLength:       storedLen,
		UncompressedLength: len(uncompressed),
	}

	playerOffset := gzipOffset + int(compressedLen)
	pr := newReader(data[playerOffset : len(data)-trailerSize])
	player, err := decodePlayer(pr)
	if err != nil {
		return nil, fmt.Errorf("fch: player section at offset %d: %w", playerOffset+pr.pos, err)
	}
	c.Player = player
	c.RemainingBytes = pr.remaining()

	trailerOffset := len(data) - trailerSize
	tr := newReader(data[trailerOffset:])
	c.Trailer.Offset = trailerOffset
	c.Trailer.Unknown = tr.u32()
	c.Trailer.Length = tr.u32()
	if c.Trailer.Length != 64 {
		return nil, fmt.Errorf("fch: unexpected trailer hash length %d", c.Trailer.Length)
	}
	c.Trailer.Hash = append([]byte(nil), tr.bytes(64)...)
	if tr.err != nil {
		return nil, tr.err
	}
	return c, nil
}

func decodePlayer(r *reader) (PlayerData, error) {
	var p PlayerData
	p.Name = r.str()
	p.PlayerID = r.u64()
	p.UnknownA = r.u32()
	p.UnknownB = r.u32()
	p.UnknownFlags = []bool{r.bool(), r.bool()}

	worldCount := r.u32()
	p.Worlds = make([]WorldData, 0, worldCount)
	for i := uint32(0); i < worldCount; i++ {
		w := WorldData{Name: r.str(), Time: r.f32()}
		w.Entries = readStatEntries(r)
		p.Worlds = append(p.Worlds, w)
	}

	p.KnownTexts = readStatEntries(r)
	p.EnemyStats = readStatEntries(r)
	p.MaterialStats = readStatEntries(r)
	p.RecipeStats = readStatEntries(r)
	p.GuardianPower = readGuardianPower(r)
	p.Inventory = readInventory(r)
	if r.err != nil {
		return p, r.err
	}
	return p, nil
}

func readStatEntries(r *reader) []StatEntry {
	count := r.u32()
	out := make([]StatEntry, 0, count)
	for i := uint32(0); i < count; i++ {
		out = append(out, StatEntry{Name: r.str(), Value: r.f32()})
	}
	return out
}

func readGuardianPower(r *reader) GuardianPower {
	g := GuardianPower{
		UnknownBool:   r.bool(),
		UnknownIntA:   r.u32(),
		UnknownIntB:   r.u32(),
		UnknownFloats: []float32{r.f32(), r.f32(), r.f32(), r.f32()},
		Name:          r.str(),
		Cooldown:      r.f32(),
		UnknownIntC:   r.u32(),
	}
	return g
}

func readInventory(r *reader) []Item {
	count := r.u32()
	out := make([]Item, 0, count)
	for i := uint32(0); i < count; i++ {
		out = append(out, Item{
			Name:        r.str(),
			Stack:       r.i32(),
			Durability:  r.f32(),
			GridX:       r.i32(),
			GridY:       r.i32(),
			Equipped:    r.bool(),
			Quality:     r.i32(),
			Variant:     r.i32(),
			CrafterID:   r.u64(),
			CrafterName: r.str(),
			UnknownTail: r.u64(),
			UnknownByte: r.byte(),
		})
	}
	return out
}

func gunzip(data []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

type reader struct {
	data []byte
	pos  int
	err  error
}

func newReader(data []byte) *reader {
	return &reader{data: data}
}

func (r *reader) remaining() int {
	if r.pos >= len(r.data) {
		return 0
	}
	return len(r.data) - r.pos
}

func (r *reader) need(n int) bool {
	if r.err != nil {
		return false
	}
	if n < 0 || r.pos+n > len(r.data) {
		r.err = io.ErrUnexpectedEOF
		return false
	}
	return true
}

func (r *reader) bytes(n int) []byte {
	if !r.need(n) {
		return nil
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b
}

func (r *reader) u32() uint32 {
	b := r.bytes(4)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}

func (r *reader) i32() int32 {
	return int32(r.u32())
}

func (r *reader) u64() uint64 {
	b := r.bytes(8)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint64(b)
}

func (r *reader) f32() float32 {
	return math.Float32frombits(r.u32())
}

func (r *reader) bool() bool {
	return r.byte() != 0
}

func (r *reader) byte() byte {
	b := r.bytes(1)
	if len(b) != 1 {
		return 0
	}
	return b[0]
}

func (r *reader) str() string {
	n := r.read7BitEncodedInt()
	if r.err != nil {
		return ""
	}
	b := r.bytes(n)
	return string(b)
}

func (r *reader) read7BitEncodedInt() int {
	var count uint32
	var shift uint
	for shift != 35 {
		b := r.bytes(1)
		if b == nil {
			return 0
		}
		count |= uint32(b[0]&0x7f) << shift
		if b[0]&0x80 == 0 {
			return int(count)
		}
		shift += 7
	}
	r.err = fmt.Errorf("fch: invalid 7-bit encoded integer")
	return 0
}
