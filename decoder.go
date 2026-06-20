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
	FileLength       uint32     `json:"fileLength"`
	Version          uint32     `json:"version"`
	PlayerStatCount  uint32     `json:"playerStatCount"`
	PlayerStatValues []float32  `json:"playerStatValues,omitempty"`
	Map              MapSection `json:"map"`
	Player           PlayerData `json:"player"`
	Trailer          Trailer    `json:"trailer"`
	RemainingBytes   int        `json:"remainingBytes"`
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
	SkillVersion  uint32        `json:"skillVersion,omitempty"`
	Skills        []Skill       `json:"skills,omitempty"`
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

type Skill struct {
	Type         int32   `json:"type"`
	Name         string  `json:"name,omitempty"`
	Level        float32 `json:"level"`
	DisplayLevel int32   `json:"displayLevel"`
	Accumulator  float32 `json:"accumulator"`
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
	c.PlayerStatCount = rd.u32()
	if rd.err != nil {
		return nil, rd.err
	}
	if int(c.FileLength)+trailerSize != len(data) {
		return nil, fmt.Errorf("fch: length header %d does not match file size %d", c.FileLength, len(data))
	}

	c.PlayerStatValues = make([]float32, int(c.PlayerStatCount))
	for i := range c.PlayerStatValues {
		c.PlayerStatValues[i] = rd.f32()
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
	readPlayerTail(r, &p)
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

func readPlayerTail(r *reader, p *PlayerData) {
	readStringList(r) // known recipes
	stationCount := r.u32()
	for i := uint32(0); i < stationCount; i++ {
		r.str()
		r.u32()
	}
	readStringList(r) // known materials
	readStringList(r) // shown tutorials
	readStringList(r) // uniques
	readStringList(r) // trophies

	biomeCount := r.u32()
	for i := uint32(0); i < biomeCount; i++ {
		r.u32()
	}

	knownTextCount := r.u32()
	for i := uint32(0); i < knownTextCount; i++ {
		r.str()
		r.str()
	}

	r.str() // beard
	r.str() // hair
	for i := 0; i < 6; i++ {
		r.f32()
	}
	r.u32() // model index

	foodCount := r.u32()
	for i := uint32(0); i < foodCount; i++ {
		r.str()
		r.f32()
	}

	p.SkillVersion = r.u32()
	skillCount := r.u32()
	p.Skills = make([]Skill, 0, skillCount)
	for i := uint32(0); i < skillCount; i++ {
		skillType := r.i32()
		level := r.f32()
		p.Skills = append(p.Skills, Skill{
			Type:         skillType,
			Name:         skillName(skillType),
			Level:        level,
			DisplayLevel: int32(math.Floor(float64(level))),
			Accumulator:  r.f32(),
		})
	}

	customDataCount := r.u32()
	for i := uint32(0); i < customDataCount; i++ {
		r.str()
		r.str()
	}
}

func readStringList(r *reader) []string {
	count := r.u32()
	out := make([]string, 0, count)
	for i := uint32(0); i < count; i++ {
		out = append(out, r.str())
	}
	return out
}

func skillName(skillType int32) string {
	switch skillType {
	case 0:
		return "None"
	case 1:
		return "Swords"
	case 2:
		return "Knives"
	case 3:
		return "Clubs"
	case 4:
		return "Polearms"
	case 5:
		return "Spears"
	case 6:
		return "Blocking"
	case 7:
		return "Axes"
	case 8:
		return "Bows"
	case 9:
		return "FireMagic"
	case 10:
		return "FrostMagic"
	case 11:
		return "Unarmed"
	case 12:
		return "Pickaxes"
	case 13:
		return "WoodCutting"
	case 100:
		return "Jump"
	case 101:
		return "Sneak"
	case 102:
		return "Run"
	case 103:
		return "Swim"
	case 105:
		return "Cooking"
	case 106:
		return "Farming"
	case 107:
		return "Crafting"
	case 108:
		return "Dodge"
	case 110:
		return "Ride"
	case 781:
		return "VL_Discipline"
	case 791:
		return "VL_Abjuration"
	case 792:
		return "VL_Alteration"
	case 793:
		return "VL_Conjuration"
	case 794:
		return "VL_Evocation"
	case 795:
		return "VL_Illusion"
	case 999:
		return "All"
	case 2015031201:
		return "PP_Alchemy"
	default:
		return ""
	}
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
		item := Item{
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
		}
		r.u64()
		r.byte()
		out = append(out, item)
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
