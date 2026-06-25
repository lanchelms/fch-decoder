package main

import (
	"fmt"
	"strconv"
	"strings"

	fch "github.com/lanchelms/fch-decoder"
)

type skillRef struct {
	skillType int32
	name      string
}

type playerStatRef struct {
	index int
	name  string
}

type inventoryItemParser struct {
	parts []string
	item  fch.Item
}

func parseInventoryItem(value string) (fch.Item, error) {
	parser := inventoryItemParser{parts: strings.Split(value, ",")}
	return parser.parse()
}

func parseInventoryName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", fmt.Errorf("remove inventory item name is required")
	}
	return name, nil
}

func (p *inventoryItemParser) parse() (fch.Item, error) {
	if err := p.parseName(); err != nil {
		return fch.Item{}, err
	}
	if err := p.parseFields(); err != nil {
		return fch.Item{}, err
	}
	return p.item, nil
}

func (p *inventoryItemParser) parseName() error {
	name := strings.TrimSpace(p.parts[0])
	if name == "" {
		return fmt.Errorf("inventory item name is required")
	}
	p.item = fch.Item{
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

func parseSkillRef(value string) (skillRef, error) {
	if n, err := strconv.ParseInt(value, 10, 32); err == nil {
		return skillRef{skillType: int32(n), name: value}, nil
	}
	skillType, ok := fch.SkillTypeByName(value)
	if !ok {
		return skillRef{}, fmt.Errorf("unknown skill %q", value)
	}
	return skillRef{skillType: skillType, name: value}, nil
}

func parsePlayerStatRef(value string) (playerStatRef, error) {
	if n, err := strconv.ParseInt(value, 10, 32); err == nil {
		if n < 0 {
			return playerStatRef{}, fmt.Errorf("invalid player stat index %d", n)
		}
		return playerStatRef{index: int(n), name: value}, nil
	}
	index, ok := fch.PlayerStatIndexByName(value)
	if !ok {
		return playerStatRef{}, fmt.Errorf("unknown player stat %q", value)
	}
	return playerStatRef{index: index, name: value}, nil
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
