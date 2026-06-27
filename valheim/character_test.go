package valheim

import (
	"strings"
	"testing"
	"time"
)

func TestNewCharacter(t *testing.T) {
	before := time.Now().Unix()
	character := NewCharacter("New Test", 123456)
	after := time.Now().Unix()

	if character.Version != supportedCharacterVersion {
		t.Fatalf("Version = %d, want %d", character.Version, supportedCharacterVersion)
	}
	if character.PlayerStatCount != uint32(len(playerStatNames)) || len(character.PlayerStats) != len(playerStatNames) {
		t.Fatalf("PlayerStats = count %d entries %d, want %d", character.PlayerStatCount, len(character.PlayerStats), len(playerStatNames))
	}
	if character.PlayerStats[0].Name != "Deaths" || character.PlayerStats[len(character.PlayerStats)-1].Name != "UsePowerDeepNorth" {
		t.Fatalf("bad player stat names: first=%q last=%q", character.PlayerStats[0].Name, character.PlayerStats[len(character.PlayerStats)-1].Name)
	}
	if string(character.Map.Raw) != string([]byte{1, 0, 0, 0, 0}) {
		t.Fatalf("Map.Raw = %v, want minimal map section", character.Map.Raw)
	}
	if character.Player.Name != "New Test" || character.Player.PlayerID != 123456 {
		t.Fatalf("player = %q/%d, want New Test/123456", character.Player.Name, character.Player.PlayerID)
	}
	if character.Player.DateCreatedUnix < before || character.Player.DateCreatedUnix > after {
		t.Fatalf("DateCreatedUnix = %d, want between %d and %d", character.Player.DateCreatedUnix, before, after)
	}
	if !character.HasPlayerData {
		t.Fatal("HasPlayerData = false, want true")
	}
	if character.Player.PlayerVersion != supportedPlayerVersion {
		t.Fatalf("PlayerVersion = %d, want %d", character.Player.PlayerVersion, supportedPlayerVersion)
	}
	if character.Player.InventoryVersion != supportedInventoryVersion {
		t.Fatalf("InventoryVersion = %d, want %d", character.Player.InventoryVersion, supportedInventoryVersion)
	}
	if character.Player.SkillVersion != supportedSkillVersion {
		t.Fatalf("SkillVersion = %d, want %d", character.Player.SkillVersion, supportedSkillVersion)
	}
}

func TestCharacterEditMethods(t *testing.T) {
	character := &Character{}

	character.AddInventoryItem(Item{Name: "Wood"})
	character.AddInventoryItem(Item{Name: "Stone"})
	if err := character.RemoveInventoryItem("Wood"); err != nil {
		t.Fatal(err)
	}
	if len(character.Player.Inventory) != 1 || character.Player.Inventory[0].Name != "Stone" {
		t.Fatalf("inventory = %+v", character.Player.Inventory)
	}

	if err := character.PutInventoryItem(Item{Name: "Resin", GridX: 0, GridY: 0}, false); err == nil {
		t.Fatal("PutInventoryItem error = nil, want occupied slot")
	}
	if err := character.PutInventoryItem(Item{Name: "Resin", GridX: 0, GridY: 0}, true); err != nil {
		t.Fatal(err)
	}
	if len(character.Player.Inventory) != 1 || character.Player.Inventory[0].Name != "Resin" {
		t.Fatalf("inventory = %+v, want replaced item", character.Player.Inventory)
	}
	if err := character.PlaceInventoryItem(Item{Name: "Feathers"}); err != nil {
		t.Fatal(err)
	}
	if item := character.Player.Inventory[1]; item.Name != "Feathers" || item.GridX != 1 || item.GridY != 0 {
		t.Fatalf("placed item = %+v, want Feathers at 1,0", item)
	}

	character.SetSkill(102, 22.75)
	character.SetSkill(102, 23.5)
	if len(character.Player.Skills) != 1 ||
		character.Player.Skills[0].Level != 23.5 ||
		character.Player.Skills[0].DisplayLevel != 23 {
		t.Fatalf("skills = %+v", character.Player.Skills)
	}

	character.UpsertEnemyStat("$enemy_greydwarf", 1)
	character.UpsertEnemyStat("$enemy_GREYDWARF", 2)
	if len(character.Player.EnemyStats) != 1 || character.Player.EnemyStats[0].Value != 2 {
		t.Fatalf("enemy stats = %+v", character.Player.EnemyStats)
	}

	character.UpsertMaterialStat("$item_wood", 50)
	if len(character.Player.MaterialStats) != 1 || character.Player.MaterialStats[0].Value != 50 {
		t.Fatalf("material stats = %+v", character.Player.MaterialStats)
	}

	if err := character.SetPlayerStat(2, "Builds", 6); err != nil {
		t.Fatal(err)
	}
	if character.PlayerStatCount != 3 || len(character.PlayerStats) != 3 || character.PlayerStats[2].Value != 6 {
		t.Fatalf("player stats = count %d entries %+v", character.PlayerStatCount, character.PlayerStats)
	}

	character.UpsertCustomData("fchedit.lastModified", "2026-06-26T12:00:00Z")
	character.UpsertCustomData("fchedit.lastModified", "2026-06-26T12:01:00Z")
	if len(character.Player.CustomData) != 1 || character.Player.CustomData[0].Value != "2026-06-26T12:01:00Z" {
		t.Fatalf("custom data = %+v", character.Player.CustomData)
	}
}

func TestCharacterQueryMethods(t *testing.T) {
	character := &Character{}
	character.Player.Inventory = []Item{
		{Name: "Wood", GridX: 0, GridY: 0},
		{Name: "Stone", GridX: 2, GridY: 1},
	}
	character.Player.Skills = []Skill{{Type: 102, Name: "Run", Level: 22.75}}
	character.Player.EnemyStats = []StatEntry{{Name: "$enemy_greydwarf", Value: 3}}
	character.Player.MaterialStats = []StatEntry{{Name: "$item_wood", Value: 50}}
	character.Player.CustomData = []TextEntry{{Key: "fchedit.lastModified", Value: "2026-06-26T12:00:00Z"}}

	if item, ok := character.InventoryItem("Wood"); !ok || item.Name != "Wood" {
		t.Fatalf("InventoryItem Wood = %+v ok=%v", item, ok)
	}
	if _, ok := character.InventoryItem("wood"); ok {
		t.Fatal("InventoryItem wood ok = true, want exact name match")
	}
	if item, ok := character.InventorySlot(2, 1); !ok || item.Name != "Stone" {
		t.Fatalf("InventorySlot 2,1 = %+v ok=%v", item, ok)
	}
	if x, y, ok := character.EmptyInventorySlot(); !ok || x != 1 || y != 0 {
		t.Fatalf("EmptyInventorySlot = %d,%d ok=%v, want 1,0 true", x, y, ok)
	}
	if skill, ok := character.Skill(102); !ok || skill.Level != 22.75 {
		t.Fatalf("Skill 102 = %+v ok=%v, want level 22.75", skill, ok)
	}
	if value, ok := character.EnemyStat("$enemy_GREYDWARF"); !ok || value != 3 {
		t.Fatalf("EnemyStat = %v ok=%v, want 3 true", value, ok)
	}
	if value, ok := character.MaterialStat("$ITEM_WOOD"); !ok || value != 50 {
		t.Fatalf("MaterialStat = %v ok=%v, want 50 true", value, ok)
	}
	if value, ok := character.CustomData("fchedit.lastModified"); !ok || value != "2026-06-26T12:00:00Z" {
		t.Fatalf("CustomData = %q ok=%v", value, ok)
	}
}

func TestPlaceInventoryItemRejectsFullInventory(t *testing.T) {
	character := &Character{}
	for y := int32(0); y < inventoryHeight; y++ {
		for x := int32(0); x < inventoryWidth; x++ {
			character.AddInventoryItem(Item{Name: "Wood", GridX: x, GridY: y})
		}
	}

	if err := character.PlaceInventoryItem(Item{Name: "Stone"}); err == nil {
		t.Fatal("PlaceInventoryItem error = nil, want full inventory error")
	}
}

func TestCharacterValidate(t *testing.T) {
	character := validCharacter()

	if err := character.Validate(); err != nil {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestCharacterValidateRejectsUnexpectedShape(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Character)
		want string
	}{
		{
			name: "character version",
			edit: func(character *Character) {
				character.Version++
			},
			want: "unsupported character version 44",
		},
		{
			name: "player version",
			edit: func(character *Character) {
				character.Player.PlayerVersion++
			},
			want: "unsupported player version 30",
		},
		{
			name: "inventory version",
			edit: func(character *Character) {
				character.Player.InventoryVersion++
			},
			want: "unsupported inventory version 107",
		},
		{
			name: "skill version",
			edit: func(character *Character) {
				character.Player.SkillVersion++
			},
			want: "unsupported skill version 3",
		},
		{
			name: "remaining bytes",
			edit: func(character *Character) {
				character.RemainingBytes = 4
			},
			want: "decoded character has 4 unread player bytes",
		},
		{
			name: "tail float count",
			edit: func(character *Character) {
				character.Player.tailFloatCount = 1
			},
			want: "fch: unsupported player tail float count 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			character := validCharacter()
			tt.edit(character)

			err := character.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestCharacterValidateEditableRejectsUnexpectedShape(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Character)
		want string
	}{
		{
			name: "trailer hash",
			edit: func(character *Character) {
				character.Trailer.HashValid = false
			},
			want: "invalid trailer hash",
		},
		{
			name: "player data",
			edit: func(character *Character) {
				character.HasPlayerData = false
			},
			want: "missing player data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			character := validCharacter()
			tt.edit(character)

			err := character.ValidateEditable()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateEditable error = %v, want %q", err, tt.want)
			}
		})
	}
}

func validCharacter() *Character {
	character := NewCharacter("Valid Test", 123456)
	character.Trailer.HashValid = true
	return character
}
