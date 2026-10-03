package valheim

import (
	"fmt"
	"math"
	"strings"

	"github.com/lanchelms/fch-decoder/binary"
)

const (
	inventoryWidth  int32 = 8
	inventoryHeight int32 = 4
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
	StatGroups       []StatGroup `json:"statGroups,omitempty"`
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
	if c.Version != 43 && c.Version != 46 {
		panic(fmt.Errorf("unsupported character version %d", c.Version))
	}
	c.PlayerStatCount = r.Uint32()

	payloadEnd := payloadStart + int(c.FileLength)
	if c.FileLength == 0 {
		payloadEnd = len(r.Data())
	}
	if int(c.PlayerStatCount) > (payloadEnd-r.Position())/4 {
		panic(fmt.Errorf("fch: player stat count %d exceeds payload size", c.PlayerStatCount))
	}

	if c.Version == 46 {
		if c.PlayerStatCount != 205 {
			panic("fch: version 46 requires 205 stats")
		}
		if r.Uint32() != 10 {
			panic("fch: version 46 requires ten stat groups")
		}
		for _, name := range statGroupNames {
			g := StatGroup{Name: name}
			g.Decode(r)
			c.StatGroups = append(c.StatGroups, g)
		}
	} else {
		c.PlayerStats = make([]StatEntry, 0, c.PlayerStatCount)
		for i := 0; i < int(c.PlayerStatCount); i++ {
			value := r.Float32()
			c.PlayerStats = append(c.PlayerStats, StatEntry{Name: playerStatName(i), Value: value})
		}

	}

	mapSection, playerOffset, err := readMapSection(r.Data(), r.Position(), payloadEnd)
	if err != nil {
		panic(err)
	}
	c.Map = mapSection

	pr := r.Slice(playerOffset, payloadEnd)
	c.Player.decodeVersion(pr, c.Version)
	c.HasPlayerData = pr.Bool()
	if c.HasPlayerData {
		c.PlayerDataLength = pr.Uint32()
		playerData := binary.NewReader(pr.Bytes(int(c.PlayerDataLength)))
		c.Player.PlayerState.Decode(playerData)
		if (c.Version == 46 && (c.Player.PlayerVersion != 33 || c.Player.InventoryVersion != 109)) || (c.Version == 43 && (c.Player.PlayerVersion != 29 || c.Player.InventoryVersion != 106)) {
			panic("fch: unsupported character/player/inventory layout")
		}
		c.Player.PlayerTail.decodeVersion(playerData, c.Player.PlayerVersion)
		if playerData.Remaining() != 0 {
			panic(fmt.Errorf("fch: %d unread embedded player bytes", playerData.Remaining()))
		}
	}
	c.RemainingBytes = pr.Remaining()
	if c.RemainingBytes != 0 {
		panic(fmt.Errorf("fch: %d unread character bytes", c.RemainingBytes))
	}
	r.SetPosition(payloadEnd)
}

func (c Character) Encode(w *binary.Writer) {
	w.Uint32(c.Version)
	if c.Version == 46 {
		w.Uint32(c.PlayerStatCount)
		w.Uint32(uint32(len(c.StatGroups)))
		for _, g := range c.StatGroups {
			g.Encode(w)
		}
	} else {
		w.Uint32(uint32(len(c.PlayerStats)))
		for _, stat := range c.PlayerStats {
			w.Float32(stat.Value)
		}
	}
	w.Bytes(c.Map.Raw)
	c.Player.encodeVersion(w, c.Version)
	c.encodePlayerData(w)
}

func (c Character) encodePlayerData(w *binary.Writer) {
	w.Bool(c.HasPlayerData)
	if !c.HasPlayerData {
		return
	}

	playerData := binary.NewWriter()
	c.Player.PlayerState.Encode(playerData)
	c.Player.PlayerTail.encodeVersion(playerData, c.Player.PlayerVersion)
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
	if c.Version != supportedCharacterVersion && c.Version != 46 {
		return fmt.Errorf("unsupported character version %d", c.Version)
	}
	if c.Version == 46 {
		if c.PlayerStatCount != 205 || len(c.StatGroups) != 10 {
			return fmt.Errorf("invalid version 46 stat groups")
		}
		for index, g := range c.StatGroups {
			if g.Name != statGroupNames[index] || len(g.Stats) != 205 || len(g.EnemyStats) != 5 {
				return fmt.Errorf("invalid stat group %q", g.Name)
			}
		}
	}
	if c.Version == 43 && c.PlayerStatCount != uint32(len(c.PlayerStats)) {
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
	if c.Version == 46 && c.Player.normalizedTailFloatCount() != 3 {
		return fmt.Errorf("version 46 requires three player tail floats")
	}
	if err := c.Player.Validate(); err != nil {
		return err
	}
	if (c.Version == 46 && (c.Player.PlayerVersion != 33 || c.Player.InventoryVersion != 109)) || (c.Version == 43 && (c.Player.PlayerVersion != 29 || c.Player.InventoryVersion != 106)) {
		return fmt.Errorf("unsupported character/player/inventory layout")
	}
	return nil
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
	for y := range inventoryHeight {
		for x := range inventoryWidth {
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
	if c.Version == 46 {
		upsertStat(&c.StatGroups[0].EnemyStats[0], name, value)
		return
	}
	upsertStat(&c.Player.EnemyStats, name, value)
}

// EnemyStat returns an enemy stat by case-insensitive name.
func (c *Character) EnemyStat(name string) (float32, bool) {
	if c.Version == 46 {
		return stat(c.StatGroups[0].EnemyStats[0], name)
	}
	return stat(c.Player.EnemyStats, name)
}

// UpsertMaterialStat updates a material stat by case-insensitive name or appends it.
func (c *Character) UpsertMaterialStat(name string, value float32) {
	if c.Version == 46 {
		upsertStat(&c.StatGroups[0].ItemsPickedUp, name, value)
		return
	}
	upsertStat(&c.Player.MaterialStats, name, value)
}

// MaterialStat returns a material stat by case-insensitive name.
func (c *Character) MaterialStat(name string) (float32, bool) {
	if c.Version == 46 {
		return stat(c.StatGroups[0].ItemsPickedUp, name)
	}
	return stat(c.Player.MaterialStats, name)
}

// SetPlayerStat sets a player stat by index and keeps PlayerStatCount synchronized.
func (c *Character) SetPlayerStat(index int, name string, value float32) error {
	if c.Version == 46 {
		if index < 0 || index >= 205 {
			return fmt.Errorf("invalid stat index %d", index)
		}
		c.StatGroups[0].Stats[index] = StatEntry{Name: currentPlayerStatNames[index], Value: value}
		return nil
	}
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
