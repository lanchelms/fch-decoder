package fch

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

//go:embed data/items.tsv
var embeddedItemCatalog string

var (
	itemCatalogOnce sync.Once
	itemCatalog     *ItemCatalog
)

// ItemCatalog holds Valheim ObjectDB item prefab metadata.
type ItemCatalog struct {
	items []ItemMetadata
	index map[string]int
}

// Items returns the embedded Valheim item catalog.
func Items() *ItemCatalog {
	var err error
	itemCatalogOnce.Do(func() {
		itemCatalog, err = parseItemCatalog(embeddedItemCatalog)
	})
	if err != nil {
		panic(err)
	}
	return itemCatalog
}

// Lookup returns metadata for a Valheim item prefab name.
func (c *ItemCatalog) Lookup(name string) (ItemMetadata, bool) {
	index, ok := c.index[name]
	if !ok {
		return ItemMetadata{}, false
	}
	return c.items[index], true
}

// List returns all known Valheim item prefab metadata sorted by name.
func (c *ItemCatalog) List() []ItemMetadata {
	return append([]ItemMetadata(nil), c.items...)
}

// Names returns all known Valheim item prefab names sorted by name.
func (c *ItemCatalog) Names() []string {
	names := make([]string, 0, len(c.items))
	for _, item := range c.items {
		names = append(names, item.Name)
	}
	return names
}

// ItemMetadata describes a Valheim ObjectDB item prefab that can be written to
// inventory records.
type ItemMetadata struct {
	Name           string  `json:"name"`
	InventoryValid bool    `json:"inventoryValid"`
	BaseDurability float32 `json:"baseDurability"`
	DurabilityStep float32 `json:"durabilityStep"`
	MaxQuality     int32   `json:"maxQuality"`
	MaxStack       int32   `json:"maxStack"`
}

// Durability returns the default full durability for the requested quality.
func (m ItemMetadata) Durability(quality int32) float32 {
	if quality < 1 {
		quality = 1
	}
	return m.BaseDurability + float32(quality-1)*m.DurabilityStep
}

func parseItemCatalog(data string) (*ItemCatalog, error) {
	rows := itemRowCount(data)
	catalog := &ItemCatalog{
		items: make([]ItemMetadata, 0, rows),
		index: make(map[string]int, rows),
	}
	for lineNumber, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		item, err := parseItemRow(line)
		if err != nil {
			return nil, fmt.Errorf("item catalog line %d: %w", lineNumber+1, err)
		}
		if _, exists := catalog.index[item.Name]; exists {
			return nil, fmt.Errorf("item catalog line %d: duplicate item %q", lineNumber+1, item.Name)
		}
		catalog.index[item.Name] = len(catalog.items)
		catalog.items = append(catalog.items, item)
	}
	return catalog, nil
}

func itemRowCount(data string) int {
	count := 0
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			count++
		}
	}
	return count
}

func parseItemRow(line string) (ItemMetadata, error) {
	fields := strings.Split(line, "\t")
	if len(fields) != 6 {
		return ItemMetadata{}, fmt.Errorf("got %d fields, want 6", len(fields))
	}
	inventoryValid, err := strconv.ParseBool(fields[1])
	if err != nil {
		return ItemMetadata{}, fmt.Errorf("inventory valid: %w", err)
	}
	baseDurability, err := parseCatalogFloat32(fields[2])
	if err != nil {
		return ItemMetadata{}, fmt.Errorf("base durability: %w", err)
	}
	durabilityStep, err := parseCatalogFloat32(fields[3])
	if err != nil {
		return ItemMetadata{}, fmt.Errorf("durability step: %w", err)
	}
	maxQuality, err := parseCatalogInt32(fields[4])
	if err != nil {
		return ItemMetadata{}, fmt.Errorf("max quality: %w", err)
	}
	maxStack, err := parseCatalogInt32(fields[5])
	if err != nil {
		return ItemMetadata{}, fmt.Errorf("max stack: %w", err)
	}
	return ItemMetadata{
		Name:           fields[0],
		InventoryValid: inventoryValid,
		BaseDurability: baseDurability,
		DurabilityStep: durabilityStep,
		MaxQuality:     maxQuality,
		MaxStack:       maxStack,
	}, nil
}

func parseCatalogFloat32(value string) (float32, error) {
	f, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return 0, err
	}
	return float32(f), nil
}

func parseCatalogInt32(value string) (int32, error) {
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(n), nil
}
