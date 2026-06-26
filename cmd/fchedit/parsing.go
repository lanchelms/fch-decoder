package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	fch "github.com/lanchelms/fch-decoder"
)

const maxSkillLevel = 100

type skillRef struct {
	skillType int32
	name      string
}

type playerStatRef struct {
	index int
	name  string
}

type inventoryItemParser struct {
	parts         []string
	metadata      fch.ItemMetadata
	item          fch.Item
	positioned    bool
	durabilitySet bool
}

func parseInventoryItem(value string) (fch.Item, bool, error) {
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

func (p *inventoryItemParser) parse() (fch.Item, bool, error) {
	if err := p.parseName(); err != nil {
		return fch.Item{}, false, err
	}
	if err := p.parseFields(); err != nil {
		return fch.Item{}, false, err
	}
	return p.item, p.positioned, nil
}

func (p *inventoryItemParser) parseName() error {
	name := strings.TrimSpace(p.parts[0])
	if name == "" {
		return fmt.Errorf("inventory item name is required")
	}
	metadata, ok := fch.Items().Lookup(name)
	if !ok {
		return fmt.Errorf("unknown inventory item %q", name)
	}
	p.metadata = metadata
	p.item = fch.Item{
		Name:       metadata.Name,
		Stack:      1,
		Durability: metadata.Durability(1),
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
		if err == nil && p.item.Stack < 1 {
			return fmt.Errorf("must be at least 1")
		}
	case "durability":
		p.item.Durability, err = parseFiniteFloat32(raw)
		if err == nil && p.item.Durability < 0 {
			return fmt.Errorf("must be nonnegative")
		}
		p.durabilitySet = err == nil
	case "pos":
		err = p.setPosition(raw)
	case "equipped":
		p.item.Equipped, err = strconv.ParseBool(raw)
	case "quality":
		p.item.Quality, err = parseInt32(raw)
		if err == nil && p.item.Quality < 1 {
			return fmt.Errorf("must be at least 1")
		}
		if err == nil && p.item.Quality > p.metadata.MaxQuality {
			return fmt.Errorf("must be at most %d for %s", p.metadata.MaxQuality, p.metadata.Name)
		}
		if err == nil && !p.durabilitySet {
			p.item.Durability = p.metadata.Durability(p.item.Quality)
		}
	case "variant":
		p.item.Variant, err = parseInt32(raw)
		if err == nil && p.item.Variant < 0 {
			return fmt.Errorf("must be nonnegative")
		}
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

func (p *inventoryItemParser) setPosition(raw string) error {
	x, y, ok := strings.Cut(raw, ":")
	if !ok || x == "" || y == "" || strings.Contains(y, ":") {
		return fmt.Errorf("must be x:y")
	}

	gridX, err := parseInt32(x)
	if err != nil {
		return err
	}
	if gridX < 0 {
		return fmt.Errorf("x must be nonnegative")
	}

	gridY, err := parseInt32(y)
	if err != nil {
		return err
	}
	if gridY < 0 {
		return fmt.Errorf("y must be nonnegative")
	}

	p.item.GridX = gridX
	p.item.GridY = gridY
	p.positioned = true
	return nil
}

func parseStatName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", fmt.Errorf("stat name is required")
	}
	return name, nil
}

func parseSkillLevel(value float32) (float32, error) {
	if err := validateFinite(value); err != nil {
		return 0, fmt.Errorf("invalid skill level: %w", err)
	}
	if value < 0 || value > maxSkillLevel {
		return 0, fmt.Errorf("invalid skill level %v: must be between 0 and %d", value, maxSkillLevel)
	}
	return value, nil
}

func parseStatValue(value float32) (float32, error) {
	if err := validateFinite(value); err != nil {
		return 0, fmt.Errorf("invalid stat value: %w", err)
	}
	if value < 0 {
		return 0, fmt.Errorf("invalid stat value %v: must be nonnegative", value)
	}
	return value, nil
}

func parseSkillRef(value string) (skillRef, error) {
	value = strings.TrimSpace(value)
	if n, err := strconv.ParseInt(value, 10, 32); err == nil {
		if n < 0 {
			return skillRef{}, fmt.Errorf("invalid skill type %d", n)
		}
		return skillRef{skillType: int32(n), name: value}, nil
	}
	skillType, ok := fch.SkillTypeByName(value)
	if !ok {
		return skillRef{}, fmt.Errorf("unknown skill %q", value)
	}
	return skillRef{skillType: skillType, name: value}, nil
}

func parsePlayerStatRef(value string) (playerStatRef, error) {
	value = strings.TrimSpace(value)
	if n, err := strconv.ParseInt(value, 10, 32); err == nil {
		if n < 0 {
			return playerStatRef{}, fmt.Errorf("invalid player stat index %d", n)
		}
		if n >= int64(len(fch.PlayerStatNames())) {
			return playerStatRef{}, fmt.Errorf("unknown player stat index %d", n)
		}
		return playerStatRef{index: int(n), name: value}, nil
	}
	index, ok := fch.PlayerStatIndexByName(value)
	if !ok {
		return playerStatRef{}, fmt.Errorf("unknown player stat %q", value)
	}
	return playerStatRef{index: index, name: value}, nil
}

func parseFiniteFloat32(value string) (float32, error) {
	f, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return 0, err
	}
	v := float32(f)
	if err := validateFinite(v); err != nil {
		return 0, err
	}
	return v, nil
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

func validateFinite(value float32) error {
	if math.IsNaN(float64(value)) {
		return fmt.Errorf("must not be NaN")
	}
	if math.IsInf(float64(value), 0) {
		return fmt.Errorf("must be finite")
	}
	return nil
}
