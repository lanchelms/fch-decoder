package fch

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"
)

const trailerSize = 72

type Character struct {
	FileLength      uint32      `json:"fileLength"`
	Version         uint32      `json:"version"`
	PlayerStatCount uint32      `json:"playerStatCount"`
	PlayerStats     []StatEntry `json:"playerStats,omitempty"`
	Map             MapSection  `json:"map"`
	Player          PlayerData  `json:"player"`
	Trailer         Trailer     `json:"trailer"`
	RemainingBytes  int         `json:"remainingBytes"`
}

type MapSection struct {
	Offset           int    `json:"offset"`
	CompressedLength uint32 `json:"compressedLength"`
	StoredLength     uint32 `json:"storedLength"`
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

	c.PlayerStats = make([]StatEntry, 0, c.PlayerStatCount)
	for i := 0; i < int(c.PlayerStatCount); i++ {
		value := rd.f32()
		c.PlayerStats = append(c.PlayerStats, StatEntry{Name: playerStatName(i), Value: value})
	}

	mapSection, playerOffset, err := readMapSection(data, rd.pos)
	if err != nil {
		return nil, err
	}
	c.Map = mapSection

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

func readMapSection(data []byte, startOffset int) (MapSection, int, error) {
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
	if gzipOffset+int(compressedLen) > len(data)-trailerSize {
		return MapSection{}, 0, fmt.Errorf("fch: invalid compressed map length %d at offset %d", compressedLen, gzipOffset)
	}

	return MapSection{
		Offset:           gzipOffset,
		CompressedLength: compressedLen,
		StoredLength:     storedLen,
	}, gzipOffset + int(compressedLen), nil
}

func decodePlayer(r *reader) (PlayerData, error) {
	var p PlayerData
	p.Name = r.str()
	p.PlayerID = r.u64()
	p.StartSeed = r.str()
	p.UsedCheats = r.bool()
	p.DateCreatedUnix = int64(r.u64())

	p.KnownWorlds = readTimedEntries(r)
	p.KnownWorldKeys = readWorldKeys(r)
	p.KnownCommands = readStatEntries(r)
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
	for range count {
		out = append(out, StatEntry{Name: r.str(), Value: r.f32()})
	}
	return out
}

func readTimedEntries(r *reader) []TimedEntry {
	count := r.u32()
	out := make([]TimedEntry, 0, count)
	for range count {
		out = append(out, TimedEntry{Name: r.str(), Seconds: r.f32()})
	}
	return out
}

func readWorldKeys(r *reader) []WorldKey {
	count := r.u32()
	out := make([]WorldKey, 0, count)
	for range count {
		raw := r.str()
		out = append(out, parseWorldKey(raw, r.f32()))
	}
	return out
}

func parseWorldKey(raw string, seconds float32) WorldKey {
	key, setting, ok := strings.Cut(raw, " ")
	if !ok {
		return WorldKey{Raw: raw, Seconds: seconds}
	}
	return WorldKey{Raw: raw, Key: key, Setting: setting, Seconds: seconds}
}

func readPlayerTail(r *reader, p *PlayerData) {
	p.KnownRecipes = r.stringList()
	stationCount := r.u32()
	p.KnownStations = make([]Station, 0, stationCount)
	for range stationCount {
		p.KnownStations = append(p.KnownStations, Station{Name: r.str(), Level: r.u32()})
	}
	p.KnownMaterials = r.stringList()
	p.ShownTutorials = r.stringList()
	p.Uniques = r.stringList()
	p.Trophies = r.stringList()

	biomeCount := r.u32()
	p.KnownBiomes = make([]uint32, 0, biomeCount)
	for range biomeCount {
		p.KnownBiomes = append(p.KnownBiomes, r.u32())
	}

	knownTextCount := r.u32()
	p.PlayerKnownTexts = make([]TextEntry, 0, knownTextCount)
	for range knownTextCount {
		p.PlayerKnownTexts = append(p.PlayerKnownTexts, TextEntry{Key: r.str(), Value: r.str()})
	}

	p.Beard = r.str()
	p.Hair = r.str()
	p.SkinColor = r.vector3()
	p.HairColor = r.vector3()
	p.ModelIndex = r.u32()

	foodCount := r.u32()
	p.Foods = make([]Food, 0, foodCount)
	for range foodCount {
		p.Foods = append(p.Foods, Food{Name: r.str(), Time: r.f32()})
	}

	p.SkillVersion = r.u32()
	skillCount := r.u32()
	p.Skills = make([]Skill, 0, skillCount)
	for range skillCount {
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
	p.CustomData = make([]TextEntry, 0, customDataCount)
	for range customDataCount {
		p.CustomData = append(p.CustomData, TextEntry{Key: r.str(), Value: r.str()})
	}
	if r.remaining() >= 8 {
		p.Stamina = r.f32()
		p.MaxEitr = r.f32()
	}
	if r.remaining() >= 4 {
		p.Eitr = r.f32()
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
		customDataCount := r.u32()
		item.CustomData = make([]TextEntry, 0, customDataCount)
		for range customDataCount {
			item.CustomData = append(item.CustomData, TextEntry{Key: r.str(), Value: r.str()})
		}
		item.WorldLevel = r.u32()
		item.PickedUp = r.bool()
		out = append(out, item)
	}
	return out
}
