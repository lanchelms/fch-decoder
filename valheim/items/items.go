package items

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
	itemCatalog     *catalog
)

// catalog holds Valheim ObjectDB item prefab metadata.
type catalog struct {
	items []Metadata
	index map[string]int
}

// Catalog returns the embedded Valheim item catalog.
func Catalog() *catalog {
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
func (c *catalog) Lookup(name string) (Metadata, bool) {
	index, ok := c.index[name]
	if !ok {
		return Metadata{}, false
	}
	return c.items[index], true
}

// List returns all known Valheim item prefab metadata sorted by name.
func (c *catalog) List() []Metadata {
	return append([]Metadata(nil), c.items...)
}

// Names returns all known Valheim item prefab names sorted by name.
func (c *catalog) Names() []string {
	names := make([]string, 0, len(c.items))
	for _, item := range c.items {
		names = append(names, item.Name)
	}
	return names
}

// Metadata describes a Valheim ObjectDB item prefab that can be written to
// inventory records.
type Metadata struct {
	Name           string   `json:"name"`
	InventoryValid bool     `json:"inventoryValid"`
	Recipes        []string `json:"recipes,omitempty"`
	BaseDurability float32  `json:"baseDurability"`
	DurabilityStep float32  `json:"durabilityStep"`
	MaxQuality     int32    `json:"maxQuality"`
	MaxStack       int32    `json:"maxStack"`
}

// Durability returns the default full durability for the requested quality.
func (m Metadata) Durability(quality int32) float32 {
	if quality < 1 {
		quality = 1
	}
	return m.BaseDurability + float32(quality-1)*m.DurabilityStep
}

func parseItemCatalog(data string) (*catalog, error) {
	rows := itemRowCount(data)
	catalog := &catalog{
		items: make([]Metadata, 0, rows),
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

func parseItemRow(line string) (Metadata, error) {
	fields := strings.Split(line, "\t")
	if len(fields) != 7 {
		return Metadata{}, fmt.Errorf("got %d fields, want 7", len(fields))
	}
	inventoryValid, err := strconv.ParseBool(fields[1])
	if err != nil {
		return Metadata{}, fmt.Errorf("inventory valid: %w", err)
	}
	recipes := parseItemRecipes(fields[2])
	baseDurability, err := parseCatalogFloat32(fields[3])
	if err != nil {
		return Metadata{}, fmt.Errorf("base durability: %w", err)
	}
	durabilityStep, err := parseCatalogFloat32(fields[4])
	if err != nil {
		return Metadata{}, fmt.Errorf("durability step: %w", err)
	}
	maxQuality, err := parseCatalogInt32(fields[5])
	if err != nil {
		return Metadata{}, fmt.Errorf("max quality: %w", err)
	}
	maxStack, err := parseCatalogInt32(fields[6])
	if err != nil {
		return Metadata{}, fmt.Errorf("max stack: %w", err)
	}
	return Metadata{
		Name:           fields[0],
		InventoryValid: inventoryValid,
		Recipes:        recipes,
		BaseDurability: baseDurability,
		DurabilityStep: durabilityStep,
		MaxQuality:     maxQuality,
		MaxStack:       maxStack,
	}, nil
}

func parseItemRecipes(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
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
