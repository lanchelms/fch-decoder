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
	Name             string        `json:"name"`
	PlayerID         uint64        `json:"playerId"`
	StartSeed        string        `json:"startSeed"`
	WorldID          uint64        `json:"worldId"`
	UnknownFlag      bool          `json:"unknownFlag"`
	Worlds           []WorldData   `json:"worlds"`
	KnownTexts       []StatEntry   `json:"knownTexts,omitempty"`
	EnemyStats       []StatEntry   `json:"enemyStats,omitempty"`
	MaterialStats    []StatEntry   `json:"materialStats,omitempty"`
	RecipeStats      []StatEntry   `json:"recipeStats,omitempty"`
	GuardianPower    GuardianPower `json:"guardianPower"`
	HasPlayerData    bool          `json:"hasPlayerData"`
	PlayerDataLength uint32        `json:"playerDataLength"`
	PlayerVersion    uint32        `json:"playerVersion"`
	MaxHealth        float32       `json:"maxHealth"`
	Health           float32       `json:"health"`
	MaxStamina       float32       `json:"maxStamina"`
	TimeSinceDeath   float32       `json:"timeSinceDeath"`
	InventoryVersion uint32        `json:"inventoryVersion"`
	Inventory        []Item        `json:"inventory,omitempty"`
	SkillVersion     uint32        `json:"skillVersion,omitempty"`
	Skills           []Skill       `json:"skills,omitempty"`
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
	Name     string  `json:"name"`
	Cooldown float32 `json:"cooldown"`
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
	p.StartSeed = r.str()
	p.WorldID = r.u64()
	p.UnknownFlag = r.bool()

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
	readPlayerState(r, &p)
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
	r.stringList() // known recipes
	stationCount := r.u32()
	for i := uint32(0); i < stationCount; i++ {
		r.str()
		r.u32()
	}
	r.stringList() // known materials
	r.stringList() // shown tutorials
	r.stringList() // uniques
	r.stringList() // trophies

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

var skillNames = map[int32]string{
	0:   "None",
	1:   "Swords",
	2:   "Knives",
	3:   "Clubs",
	4:   "Polearms",
	5:   "Spears",
	6:   "Blocking",
	7:   "Axes",
	8:   "Bows",
	9:   "ElementalMagic",
	10:  "BloodMagic",
	11:  "Unarmed",
	12:  "Pickaxes",
	13:  "WoodCutting",
	14:  "Crossbows",
	100: "Jump",
	101: "Sneak",
	102: "Run",
	103: "Swim",
	104: "Fishing",
	105: "Cooking",
	106: "Farming",
	107: "Crafting",
	108: "Dodge",
	110: "Ride",
	999: "All",
}

func skillName(skillType int32) string {
	return skillNames[skillType]
}

func readPlayerState(r *reader, p *PlayerData) {
	p.HasPlayerData = r.bool()
	p.PlayerDataLength = r.u32()
	p.PlayerVersion = r.u32()
	p.MaxHealth = r.f32()
	p.Health = r.f32()
	p.MaxStamina = r.f32()
	p.TimeSinceDeath = r.f32()
	p.GuardianPower = GuardianPower{
		Name:     r.str(),
		Cooldown: r.f32(),
	}
	p.InventoryVersion = r.u32()
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
