package fch

import (
	"strings"
	"testing"
)

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
			name: "trailer hash",
			edit: func(character *Character) {
				character.Trailer.HashValid = false
			},
			want: "invalid trailer hash",
		},
		{
			name: "player data",
			edit: func(character *Character) {
				character.Player.HasPlayerData = false
			},
			want: "missing player data",
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

func validCharacter() *Character {
	return &Character{
		Version: supportedCharacterVersion,
		Player: PlayerData{
			HasPlayerData:    true,
			PlayerVersion:    supportedPlayerVersion,
			InventoryVersion: supportedInventoryVersion,
			SkillVersion:     supportedSkillVersion,
		},
		Trailer: Trailer{HashValid: true},
	}
}
