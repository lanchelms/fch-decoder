package fch

import (
	"bytes"
	"fmt"
	"math"
	"strings"
)

const (
	inventoryWidth  = 8
	inventoryHeight = 4
)

const (
	supportedCharacterVersion = 43
	supportedPlayerVersion    = 29
	supportedInventoryVersion = 106
	supportedSkillVersion     = 2
)

type Character struct {
	FileLength       uint32      `json:"fileLength"`
	Version          uint32      `json:"version"`
	PlayerStatCount  uint32      `json:"playerStatCount"`
	PlayerStats      []StatEntry `json:"playerStats,omitempty"`
	Map              MapSection  `json:"map"`
	HasPlayerData    bool        `json:"hasPlayerData"`
	PlayerDataLength uint32      `json:"playerDataLength"`
	Player           Player      `json:"player"`
	Trailer          Trailer     `json:"trailer"`
	RemainingBytes   int         `json:"remainingBytes"`
}

func NewCharacter(name string, playerID uint64) *Character {
	playerStats := NewPlayerStats()
	return &Character{
		Version:         supportedCharacterVersion,
		PlayerStatCount: uint32(len(playerStats)),
		PlayerStats:     playerStats,
		Map:             MapSection{Raw: []byte{1, 0, 0, 0, 0}},
		HasPlayerData:   true,
		Player:          NewPlayer(name, playerID),
	}
}

func (c *Character) Decode(r *Reader) {
	c.FileLength = r.u32()
	c.Version = r.u32()
	c.PlayerStatCount = r.u32()

	payloadEnd := fileLengthSize + int(c.FileLength)
	if int(c.PlayerStatCount) > (payloadEnd-r.pos)/4 {
		panic(fmt.Errorf("fch: player stat count %d exceeds payload size", c.PlayerStatCount))
	}

	c.PlayerStats = make([]StatEntry, 0, c.PlayerStatCount)
	for i := 0; i < int(c.PlayerStatCount); i++ {
		value := r.f32()
		c.PlayerStats = append(c.PlayerStats, StatEntry{Name: playerStatName(i), Value: value})
	}

	mapSection, playerOffset, err := readMapSection(r.data, r.pos, payloadEnd)
	if err != nil {
		panic(err)
	}
	c.Map = mapSection

	pr := NewReader(r.data[playerOffset:payloadEnd])
	c.Player.Decode(pr)
	c.HasPlayerData = pr.bool()
	if c.HasPlayerData {
		c.PlayerDataLength = pr.u32()
		c.Player.PlayerState.Decode(pr)
		c.Player.PlayerTail.Decode(pr)
	}
	c.RemainingBytes = pr.remaining()
	r.pos = payloadEnd

	c.Trailer.Offset = payloadEnd
	c.Trailer.Length = r.u32()
	if c.Trailer.Length != payloadHashSize {
		panic(fmt.Errorf("fch: unexpected trailer hash length %d", c.Trailer.Length))
	}
	c.Trailer.Hash = append([]byte(nil), r.bytes(payloadHashSize)...)
	c.Trailer.HashValid = bytes.Equal(currentPayloadHash(r.data, c.FileLength), c.Trailer.Hash)
}

func (c Character) Encode(w *Writer) {
	payload := NewWriter()
	c.encodePayload(payload)
	payloadBytes := payload.Data()
	if len(payloadBytes) > math.MaxUint32 {
		panic(fmt.Errorf("fch: payload too large: %d bytes", len(payloadBytes)))
	}

	w.u32(uint32(len(payloadBytes)))
	w.bytes(payloadBytes)
	w.u32(payloadHashSize)
	w.bytes(payloadHash(payloadBytes))
}

func (c Character) encodePayload(w *Writer) {
	w.u32(c.Version)
	w.u32(uint32(len(c.PlayerStats)))
	for _, stat := range c.PlayerStats {
		w.f32(stat.Value)
	}
	w.bytes(c.Map.Raw)
	c.Player.Encode(w)
	c.encodePlayerData(w)
}

func (c Character) encodePlayerData(w *Writer) {
	w.bool(c.HasPlayerData)
	if !c.HasPlayerData {
		return
	}

	playerData := NewWriter()
	c.Player.PlayerState.Encode(playerData)
	c.Player.PlayerTail.Encode(playerData)
	data := playerData.Data()
	if len(data) > math.MaxUint32 {
		panic(fmt.Errorf("fch: player data too large: %d bytes", len(data)))
	}
	w.u32(uint32(len(data)))
	w.bytes(data)
}

// Validate verifies that the character is internally consistent and safe to encode.
func (c *Character) Validate() error {
	if c == nil {
		return fmt.Errorf("fch: cannot encode nil character")
	}
	if c.Version != supportedCharacterVersion {
		return fmt.Errorf("unsupported character version %d", c.Version)
	}
	if c.PlayerStatCount != uint32(len(c.PlayerStats)) {
		return fmt.Errorf("fch: player stat count %d does not match %d stats", c.PlayerStatCount, len(c.PlayerStats))
	}
	if len(c.Map.Raw) == 0 {
		return fmt.Errorf("fch: cannot encode character without raw map section")
	}
	if c.RemainingBytes != 0 {
		return fmt.Errorf("decoded character has %d unread player bytes", c.RemainingBytes)
	}
	if !c.HasPlayerData {
		return nil
	}
	return c.Player.Validate()
}

// ValidateEditable verifies that the character matches the decoded file shape this package can safely edit.
func (c *Character) ValidateEditable() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if !c.Trailer.HashValid {
		return fmt.Errorf("invalid trailer hash")
	}
	if !c.HasPlayerData {
		return fmt.Errorf("missing player data")
	}
	return nil
}

// AddInventoryItem appends item to the character inventory.
func (c *Character) AddInventoryItem(item Item) {
	c.Player.Inventory = append(c.Player.Inventory, item)
}

// CreditCraftedItem credits character as crafter when item is a recipe output
// and does not already have explicit crafter metadata.
func (c *Character) CreditCraftedItem(item Item) Item {
	metadata, ok := Items().Lookup(item.Name)
	if !ok || len(metadata.Recipes) == 0 || item.CrafterID != 0 || item.CrafterName != "" {
		return item
	}
	item.CrafterID = c.Player.PlayerID
	item.CrafterName = c.Player.Name
	return item
}

// PutInventoryItem adds item unless its grid slot is occupied. When replace is
// true, the item at the same grid slot is overwritten.
func (c *Character) PutInventoryItem(item Item, replace bool) error {
	for i, existing := range c.Player.Inventory {
		if existing.GridX == item.GridX && existing.GridY == item.GridY {
			if !replace {
				return fmt.Errorf("inventory slot %d,%d is occupied by %q", item.GridX, item.GridY, existing.Name)
			}
			c.Player.Inventory[i] = item
			return nil
		}
	}
	c.AddInventoryItem(item)
	return nil
}

// PlaceInventoryItem adds item to the first empty normal inventory slot.
func (c *Character) PlaceInventoryItem(item Item) error {
	for y := int32(0); y < inventoryHeight; y++ {
		for x := int32(0); x < inventoryWidth; x++ {
			if !c.inventorySlotOccupied(x, y) {
				item.GridX = x
				item.GridY = y
				c.AddInventoryItem(item)
				return nil
			}
		}
	}
	return fmt.Errorf("inventory has no empty slots")
}

func (c *Character) inventorySlotOccupied(x int32, y int32) bool {
	for _, item := range c.Player.Inventory {
		if item.GridX == x && item.GridY == y {
			return true
		}
	}
	return false
}

// RemoveInventoryItem removes the first inventory item with an exact name match.
func (c *Character) RemoveInventoryItem(name string) error {
	for i, item := range c.Player.Inventory {
		if item.Name == name {
			c.Player.Inventory = append(c.Player.Inventory[:i], c.Player.Inventory[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("inventory item %q not found", name)
}

// SetSkill updates an existing skill or appends a new skill record.
func (c *Character) SetSkill(skillType int32, level float32) {
	displayLevel := int32(math.Floor(float64(level)))
	for i := range c.Player.Skills {
		if c.Player.Skills[i].Type == skillType {
			c.Player.Skills[i].Level = level
			c.Player.Skills[i].DisplayLevel = displayLevel
			return
		}
	}
	c.Player.Skills = append(c.Player.Skills, Skill{
		Type:         skillType,
		Name:         skillName(skillType),
		Level:        level,
		DisplayLevel: displayLevel,
	})
}

// UpsertEnemyStat updates an enemy stat by case-insensitive name or appends it.
func (c *Character) UpsertEnemyStat(name string, value float32) {
	upsertStat(&c.Player.EnemyStats, name, value)
}

// UpsertMaterialStat updates a material stat by case-insensitive name or appends it.
func (c *Character) UpsertMaterialStat(name string, value float32) {
	upsertStat(&c.Player.MaterialStats, name, value)
}

// SetPlayerStat sets a player stat by index and keeps PlayerStatCount synchronized.
func (c *Character) SetPlayerStat(index int, name string, value float32) error {
	if index < 0 {
		return fmt.Errorf("invalid player stat index %d", index)
	}
	for len(c.PlayerStats) <= index {
		c.PlayerStats = append(c.PlayerStats, StatEntry{})
	}
	c.PlayerStats[index] = StatEntry{Name: name, Value: value}
	c.PlayerStatCount = uint32(len(c.PlayerStats))
	return nil
}

// UpsertCustomData updates player custom data by key or appends it.
func (c *Character) UpsertCustomData(key string, value string) {
	for i := range c.Player.CustomData {
		if c.Player.CustomData[i].Key == key {
			c.Player.CustomData[i].Value = value
			return
		}
	}
	c.Player.CustomData = append(c.Player.CustomData, TextEntry{Key: key, Value: value})
}

func upsertStat(entries *[]StatEntry, name string, value float32) {
	for i := range *entries {
		if strings.EqualFold((*entries)[i].Name, name) {
			(*entries)[i].Value = value
			return
		}
	}
	*entries = append(*entries, StatEntry{Name: name, Value: value})
}
