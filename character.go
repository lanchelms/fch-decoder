package fch

import (
	"fmt"
	"strings"
)

const (
	inventoryWidth  = 8
	inventoryHeight = 4
)

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

// AddInventoryItem appends item to the character inventory.
func (c *Character) AddInventoryItem(item Item) {
	c.Player.Inventory = append(c.Player.Inventory, item)
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

// SetSkillLevel updates an existing skill level or appends a new skill record.
func (c *Character) SetSkillLevel(skillType int32, name string, level float32) {
	for i := range c.Player.Skills {
		if c.Player.Skills[i].Type == skillType {
			c.Player.Skills[i].Level = level
			return
		}
	}
	c.Player.Skills = append(c.Player.Skills, Skill{
		Type:  skillType,
		Name:  name,
		Level: level,
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

func upsertStat(entries *[]StatEntry, name string, value float32) {
	for i := range *entries {
		if strings.EqualFold((*entries)[i].Name, name) {
			(*entries)[i].Value = value
			return
		}
	}
	*entries = append(*entries, StatEntry{Name: name, Value: value})
}
