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

type MapSection struct {
	Offset           int    `json:"offset"`
	CompressedLength uint32 `json:"compressedLength"`
	StoredLength     uint32 `json:"storedLength"`
	Raw              []byte `json:"-"`
}

type Trailer struct {
	Offset    int    `json:"offset"`
	Length    uint32 `json:"length"`
	Hash      []byte `json:"hash"`
	HashValid bool   `json:"hashValid"`
}

type PlayerData struct {
	Name             string        `json:"name"`
	PlayerID         uint64        `json:"playerId"`
	StartSeed        string        `json:"startSeed"`
	UsedCheats       bool          `json:"usedCheats"`
	DateCreatedUnix  int64         `json:"dateCreatedUnix"`
	KnownWorlds      []TimedEntry  `json:"knownWorlds,omitempty"`
	KnownWorldKeys   []WorldKey    `json:"knownWorldKeys,omitempty"`
	KnownCommands    []StatEntry   `json:"-"`
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
	Stamina          float32       `json:"stamina"`
	MaxEitr          float32       `json:"maxEitr"`
	Eitr             float32       `json:"eitr"`
	TimeSinceDeath   float32       `json:"timeSinceDeath"`
	InventoryVersion uint32        `json:"inventoryVersion"`
	Inventory        []Item        `json:"inventory,omitempty"`
	KnownRecipes     []string      `json:"knownRecipes,omitempty"`
	KnownStations    []Station     `json:"knownStations,omitempty"`
	KnownMaterials   []string      `json:"knownMaterials,omitempty"`
	ShownTutorials   []string      `json:"-"`
	Uniques          []string      `json:"uniques,omitempty"`
	Trophies         []string      `json:"trophies,omitempty"`
	KnownBiomes      []uint32      `json:"knownBiomes,omitempty"`
	PlayerKnownTexts []TextEntry   `json:"-"`
	Beard            string        `json:"beard,omitempty"`
	Hair             string        `json:"hair,omitempty"`
	SkinColor        Vector3       `json:"skinColor"`
	HairColor        Vector3       `json:"hairColor"`
	ModelIndex       uint32        `json:"modelIndex"`
	Foods            []Food        `json:"foods,omitempty"`
	SkillVersion     uint32        `json:"skillVersion,omitempty"`
	Skills           []Skill       `json:"skills,omitempty"`
	CustomData       []TextEntry   `json:"customData,omitempty"`
	tailFloatCount   int
}

type StatEntry struct {
	Name  string  `json:"name"`
	Value float32 `json:"value"`
}

type TimedEntry struct {
	Name    string  `json:"name"`
	Seconds float32 `json:"seconds"`
}

type WorldKey struct {
	Raw     string  `json:"raw"`
	Key     string  `json:"key,omitempty"`
	Setting string  `json:"setting,omitempty"`
	Seconds float32 `json:"seconds"`
}

type TextEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Station struct {
	Name  string `json:"name"`
	Level uint32 `json:"level"`
}

type Food struct {
	Name string  `json:"name"`
	Time float32 `json:"time"`
}

type Vector3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
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
	Name        string      `json:"name"`
	Stack       int32       `json:"stack"`
	Durability  float32     `json:"durability"`
	GridX       int32       `json:"gridX"`
	GridY       int32       `json:"gridY"`
	Equipped    bool        `json:"equipped"`
	Quality     int32       `json:"quality"`
	Variant     int32       `json:"variant"`
	CrafterID   uint64      `json:"crafterId"`
	CrafterName string      `json:"crafterName"`
	CustomData  []TextEntry `json:"customData,omitempty"`
	WorldLevel  uint32      `json:"worldLevel"`
	PickedUp    bool        `json:"pickedUp"`
}

func Decode(r io.Reader) (*Character, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return DecodeBytes(data)
}

func DecodeBytes(data []byte) (*Character, error) {
	if len(data) < fileOverhead+16 {
		return nil, fmt.Errorf("fch: file too short: %d bytes", len(data))
	}

	rd := newReader(data)
	c := &Character{}
	c.FileLength = rd.u32()
	c.Version = rd.u32()
	c.PlayerStatCount = rd.u32()
	payloadEnd := fileLengthSize + int(c.FileLength)
	if int(c.FileLength)+fileOverhead != len(data) {
		return nil, fmt.Errorf("fch: length header %d does not match file size %d", c.FileLength, len(data))
	}

	c.PlayerStats = make([]StatEntry, 0, c.PlayerStatCount)
	for i := 0; i < int(c.PlayerStatCount); i++ {
		value := rd.f32()
		c.PlayerStats = append(c.PlayerStats, StatEntry{Name: playerStatName(i), Value: value})
	}

	mapSection, playerOffset, err := readMapSection(data, rd.pos, payloadEnd)
	if err != nil {
		return nil, err
	}
	c.Map = mapSection

	pr := newReader(data[playerOffset:payloadEnd])
	player, err := decodePlayer(pr)
	if err != nil {
		return nil, fmt.Errorf("fch: player section at offset %d: %w", playerOffset+pr.pos, err)
	}
	c.Player = player
	c.RemainingBytes = pr.remaining()

	trailerOffset := payloadEnd
	tr := newReader(data[trailerOffset:])
	c.Trailer.Offset = trailerOffset
	c.Trailer.Length = tr.u32()
	if c.Trailer.Length != 64 {
		return nil, fmt.Errorf("fch: unexpected trailer hash length %d", c.Trailer.Length)
	}
	c.Trailer.Hash = append([]byte(nil), tr.bytes(64)...)
	c.Trailer.HashValid = bytes.Equal(currentPayloadHash(data, c.FileLength), c.Trailer.Hash)
	return c, nil
}

func currentPayloadHash(data []byte, payloadLen uint32) []byte {
	return payloadHash(data[fileLengthSize : fileLengthSize+int(payloadLen)])
}

func readMapSection(data []byte, startOffset int, payloadEnd int) (MapSection, int, error) {
	gzipOffset := bytes.Index(data[startOffset:], []byte{0x1f, 0x8b, 0x08})
	if gzipOffset < 0 {
		return MapSection{}, 0, fmt.Errorf("fch: gzip map block not found")
	}
	gzipOffset += startOffset
	if gzipOffset < 12 {
		return MapSection{}, 0, fmt.Errorf("fch: gzip map block starts too early")
	}

	storedLen := binary.LittleEndian.Uint32(data[gzipOffset-12 : gzipOffset-8])
	compressedLen := binary.LittleEndian.Uint32(data[gzipOffset-4 : gzipOffset])
	if gzipOffset+int(compressedLen) > payloadEnd {
		return MapSection{}, 0, fmt.Errorf("fch: invalid compressed map length %d at offset %d", compressedLen, gzipOffset)
	}

	return MapSection{
		Offset:           gzipOffset,
		CompressedLength: compressedLen,
		StoredLength:     storedLen,
		Raw:              append([]byte(nil), data[startOffset:gzipOffset+int(compressedLen)]...),
	}, gzipOffset + int(compressedLen), nil
}

func decodePlayer(r *reader) (PlayerData, error) {
	var p PlayerData
	p.Name = r.str()
	p.PlayerID = r.u64()
	p.StartSeed = r.str()
	p.UsedCheats = r.bool()
	p.DateCreatedUnix = int64(r.u64())

	p.KnownWorlds = readList(r, timedEntry)
	p.KnownWorldKeys = readList(r, worldKey)
	p.KnownCommands = readList(r, statEntry)
	p.EnemyStats = readList(r, statEntry)
	p.MaterialStats = readList(r, statEntry)
	p.RecipeStats = readList(r, statEntry)
	readPlayerState(r, &p)
	p.Inventory = readInventory(r)
	readPlayerTail(r, &p)
	return p, nil
}

func readPlayerTail(r *reader, p *PlayerData) {
	p.KnownRecipes = readList(r, str)
	p.KnownStations = readList(r, station)
	p.KnownMaterials = readList(r, str)
	p.ShownTutorials = readList(r, str)
	p.Uniques = readList(r, str)
	p.Trophies = readList(r, str)

	p.KnownBiomes = readList(r, biome)

	p.PlayerKnownTexts = readList(r, textEntry)

	p.Beard = r.str()
	p.Hair = r.str()
	p.SkinColor = r.vector3()
	p.HairColor = r.vector3()
	p.ModelIndex = r.u32()

	p.Foods = readList(r, food)

	p.SkillVersion = r.u32()
	p.Skills = readList(r, skill)

	p.CustomData = readList(r, textEntry)
	if r.remaining() >= 8 {
		p.Stamina = r.f32()
		p.MaxEitr = r.f32()
		p.tailFloatCount = 2
	}
	if r.remaining() >= 4 {
		p.Eitr = r.f32()
		p.tailFloatCount = 3
	}
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
	for range count {
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
		item.CustomData = readList(r, textEntry)
		item.WorldLevel = r.u32()
		item.PickedUp = r.bool()
		out = append(out, item)
	}
	return out
}
