package fch

import (
	"fmt"
	"strconv"
	"strings"
)

type StatAssignment struct {
	Name  string
	Value float32
}

type inventoryItemParser struct {
	parts []string
	item  Item
}

// ParseInventoryItem parses a structured inventory item specification.
func ParseInventoryItem(value string) (Item, error) {
	parser := inventoryItemParser{parts: strings.Split(value, ",")}
	return parser.parse()
}

func (p *inventoryItemParser) parse() (Item, error) {
	if err := p.parseName(); err != nil {
		return Item{}, err
	}
	if err := p.parseFields(); err != nil {
		return Item{}, err
	}
	return p.item, nil
}

func (p *inventoryItemParser) parseName() error {
	name := strings.TrimSpace(p.parts[0])
	if name == "" {
		return fmt.Errorf("inventory item name is required")
	}
	p.item = Item{
		Name:       name,
		Stack:      1,
		Durability: 1,
		Quality:    1,
		PickedUp:   true,
	}
	return nil
}

func (p *inventoryItemParser) parseFields() error {
	for _, part := range p.parts[1:] {
		if err := p.parseField(part); err != nil {
			return err
		}
	}
	return nil
}

func (p *inventoryItemParser) parseField(part string) error {
	key, raw, ok := strings.Cut(part, "=")
	if !ok {
		return fmt.Errorf("invalid inventory item field %q", part)
	}
	key = strings.TrimSpace(key)
	raw = strings.TrimSpace(raw)
	if err := p.setField(key, raw); err != nil {
		return fmt.Errorf("invalid inventory item field %q: %w", key, err)
	}
	return nil
}

func (p *inventoryItemParser) setField(key string, raw string) error {
	var err error
	switch key {
	case "stack":
		p.item.Stack, err = parseInt32(raw)
	case "durability":
		p.item.Durability, err = parseFloat32(raw)
	case "grid-x":
		p.item.GridX, err = parseInt32(raw)
	case "grid-y":
		p.item.GridY, err = parseInt32(raw)
	case "equipped":
		p.item.Equipped, err = strconv.ParseBool(raw)
	case "quality":
		p.item.Quality, err = parseInt32(raw)
	case "variant":
		p.item.Variant, err = parseInt32(raw)
	case "crafter-id":
		p.item.CrafterID, err = strconv.ParseUint(raw, 10, 64)
	case "crafter-name":
		p.item.CrafterName = raw
	case "world-level":
		p.item.WorldLevel, err = parseUint32(raw)
	case "picked-up":
		p.item.PickedUp, err = strconv.ParseBool(raw)
	default:
		return fmt.Errorf("unknown inventory item field %q", key)
	}
	return err
}

// ParseStatAssignment parses a name=value assignment with a float32 value.
func ParseStatAssignment(value string) (StatAssignment, error) {
	name, raw, ok := strings.Cut(value, "=")
	if !ok {
		return StatAssignment{}, fmt.Errorf("expected name=value, got %q", value)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return StatAssignment{}, fmt.Errorf("assignment name is required")
	}
	amount, err := parseFloat32(strings.TrimSpace(raw))
	if err != nil {
		return StatAssignment{}, err
	}
	return StatAssignment{Name: name, Value: amount}, nil
}

// ParseSkillType resolves either a numeric skill type or a known skill name.
func ParseSkillType(value string) (int32, string, error) {
	if n, err := strconv.ParseInt(value, 10, 32); err == nil {
		return int32(n), value, nil
	}
	skillType, ok := SkillTypeByName(value)
	if !ok {
		return 0, "", fmt.Errorf("unknown skill %q", value)
	}
	return skillType, value, nil
}

// ParsePlayerStatIndex resolves either a numeric player stat index or a known player stat name.
func ParsePlayerStatIndex(value string) (int, string, error) {
	if n, err := strconv.ParseInt(value, 10, 32); err == nil {
		if n < 0 {
			return 0, "", fmt.Errorf("invalid player stat index %d", n)
		}
		return int(n), value, nil
	}
	index, ok := PlayerStatIndexByName(value)
	if !ok {
		return 0, "", fmt.Errorf("unknown player stat %q", value)
	}
	return index, value, nil
}

func parseFloat32(value string) (float32, error) {
	f, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return 0, err
	}
	return float32(f), nil
}

func parseInt32(value string) (int32, error) {
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(n), nil
}

func parseUint32(value string) (uint32, error) {
	n, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(n), nil
}
