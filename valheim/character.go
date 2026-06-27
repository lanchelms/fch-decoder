package valheim

import (
	"fmt"
	"math"
	"strings"

	"github.com/lanchelms/fch-decoder/binary"
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
	Map              Map         `json:"map"`
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
		Map:             Map{Raw: []byte{1, 0, 0, 0, 0}},
		HasPlayerData:   true,
		Player:          NewPlayer(name, playerID),
	}
}

func (c *Character) Decode(r *binary.Reader) {
	payloadStart := r.Position()
	c.Version = r.Uint32()
	c.PlayerStatCount = r.Uint32()

	payloadEnd := payloadStart + int(c.FileLength)
	if c.FileLength == 0 {
		payloadEnd = len(r.Data())
	}
	if int(c.PlayerStatCount) > (payloadEnd-r.Position())/4 {
		panic(fmt.Errorf("fch: player stat count %d exceeds payload size", c.PlayerStatCount))
	}

	c.PlayerStats = make([]StatEntry, 0, c.PlayerStatCount)
	for i := 0; i < int(c.PlayerStatCount); i++ {
		value := r.Float32()
		c.PlayerStats = append(c.PlayerStats, StatEntry{Name: playerStatName(i), Value: value})
	}

	mapSection, playerOffset, err := readMapSection(r.Data(), r.Position(), payloadEnd)
	if err != nil {
		panic(err)
	}
	c.Map = mapSection

	pr := r.Slice(playerOffset, payloadEnd)
	c.Player.Decode(pr)
	c.HasPlayerData = pr.Bool()
	if c.HasPlayerData {
		c.PlayerDataLength = pr.Uint32()
		c.Player.PlayerState.Decode(pr)
		c.Player.PlayerTail.Decode(pr)
	}
	c.RemainingBytes = pr.Remaining()
	r.SetPosition(payloadEnd)
}

func (c Character) Encode(w *binary.Writer) {
	w.Uint32(c.Version)
	w.Uint32(uint32(len(c.PlayerStats)))
	for _, stat := range c.PlayerStats {
		w.Float32(stat.Value)
	}
	w.Bytes(c.Map.Raw)
	c.Player.Encode(w)
	c.encodePlayerData(w)
}

func (c Character) encodePlayerData(w *binary.Writer) {
	w.Bool(c.HasPlayerData)
	if !c.HasPlayerData {
		return
	}

	playerData := binary.NewWriter()
	c.Player.PlayerState.Encode(playerData)
	c.Player.PlayerTail.Encode(playerData)
	data := playerData.Data()
	if len(data) > math.MaxUint32 {
		panic(fmt.Errorf("fch: player data too large: %d bytes", len(data)))
	}
	w.Uint32(uint32(len(data)))
	w.Bytes(data)
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

// InventoryItem returns the first inventory item with an exact name match.
func (c *Character) InventoryItem(name string) (*Item, bool) {
	for i := range c.Player.Inventory {
		if c.Player.Inventory[i].Name == name {
			return &c.Player.Inventory[i], true
		}
	}
	return nil, false
}

// InventorySlot returns the inventory item at a grid slot.
func (c *Character) InventorySlot(x, y int32) (*Item, bool) {
	for i := range c.Player.Inventory {
		if c.Player.Inventory[i].GridX == x && c.Player.Inventory[i].GridY == y {
			return &c.Player.Inventory[i], true
		}
	}
	return nil, false
}

// EmptyInventorySlot returns the first empty normal inventory slot.
func (c *Character) EmptyInventorySlot() (int32, int32, bool) {
	for y := int32(0); y < inventoryHeight; y++ {
		for x := int32(0); x < inventoryWidth; x++ {
			if _, occupied := c.InventorySlot(x, y); !occupied {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

// PutInventoryItem adds item unless its grid slot is occupied. When replace is
// true, the item at the same grid slot is overwritten.
func (c *Character) PutInventoryItem(item Item, replace bool) error {
	existing, occupied := c.InventorySlot(item.GridX, item.GridY)
	if !occupied {
		c.AddInventoryItem(item)
		return nil
	}
	if !replace {
		return fmt.Errorf("inventory slot %d,%d is occupied by %q", item.GridX, item.GridY, existing.Name)
	}
	*existing = item
	return nil
}

// PlaceInventoryItem adds item to the first empty normal inventory slot.
func (c *Character) PlaceInventoryItem(item Item) error {
	x, y, ok := c.EmptyInventorySlot()
	if !ok {
		return fmt.Errorf("inventory has no empty slots")
	}
	item.GridX = x
	item.GridY = y
	c.AddInventoryItem(item)
	return nil
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
	for i := range c.Player.Skills {
		if c.Player.Skills[i].Type == skillType {
			c.Player.Skills[i].Level = level
			c.Player.Skills[i].DisplayLevel = c.Player.Skills[i].displayLevel()
			return
		}
	}
	c.Player.Skills = append(c.Player.Skills, Skill{
		Type:         skillType,
		Name:         skillName(skillType),
		Level:        level,
		DisplayLevel: int32(math.Floor(float64(level))),
	})
}

// Skill returns the saved skill by type.
func (c *Character) Skill(skillType int32) (Skill, bool) {
	for _, skill := range c.Player.Skills {
		if skill.Type == skillType {
			return skill, true
		}
	}
	return Skill{}, false
}

// UpsertEnemyStat updates an enemy stat by case-insensitive name or appends it.
func (c *Character) UpsertEnemyStat(name string, value float32) {
	upsertStat(&c.Player.EnemyStats, name, value)
}

// EnemyStat returns an enemy stat by case-insensitive name.
func (c *Character) EnemyStat(name string) (float32, bool) {
	return stat(c.Player.EnemyStats, name)
}

// UpsertMaterialStat updates a material stat by case-insensitive name or appends it.
func (c *Character) UpsertMaterialStat(name string, value float32) {
	upsertStat(&c.Player.MaterialStats, name, value)
}

// MaterialStat returns a material stat by case-insensitive name.
func (c *Character) MaterialStat(name string) (float32, bool) {
	return stat(c.Player.MaterialStats, name)
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

// CustomData returns player custom data by exact key.
func (c *Character) CustomData(key string) (string, bool) {
	for _, entry := range c.Player.CustomData {
		if entry.Key == key {
			return entry.Value, true
		}
	}
	return "", false
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

func stat(entries []StatEntry, name string) (float32, bool) {
	for _, entry := range entries {
		if strings.EqualFold(entry.Name, name) {
			return entry.Value, true
		}
	}
	return 0, false
}
