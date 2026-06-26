package fch

import (
	"bytes"
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
	if character.PlayerStatCount != 105 || len(character.PlayerStats) != 105 {
		t.Fatalf("PlayerStats = count %d entries %d, want 105", character.PlayerStatCount, len(character.PlayerStats))
	}
	if string(character.Map.Raw) != string(minimalMapSection()) {
		t.Fatalf("Map.Raw = %v, want %v", character.Map.Raw, minimalMapSection())
	}
	if character.Player.Name != "New Test" || character.Player.PlayerID != 123456 {
		t.Fatalf("player = %q/%d, want New Test/123456", character.Player.Name, character.Player.PlayerID)
	}
	if character.Player.DateCreatedUnix < before || character.Player.DateCreatedUnix > after {
		t.Fatalf("DateCreatedUnix = %d, want between %d and %d", character.Player.DateCreatedUnix, before, after)
	}
	if character.Player.HasPlayerData {
		t.Fatal("HasPlayerData = true, want false")
	}
}

func FuzzNewCharacterEncode(f *testing.F) {
	f.Add("New Test", uint64(123456))
	f.Add("", uint64(0))
	f.Add("Fenris Bueller", uint64(111111))
	f.Add("null\x00byte", uint64(1<<63))

	f.Fuzz(func(t *testing.T, name string, playerID uint64) {
		if len(name) > 4096 {
			return
		}

		character := NewCharacter(name, playerID)
		encoded, err := EncodeBytes(character)
		if err != nil {
			t.Fatalf("EncodeBytes(NewCharacter(%q, %d)) error = %v", name, playerID, err)
		}

		decoded, err := DecodeBytes(encoded)
		if err != nil {
			t.Fatalf("DecodeBytes after NewCharacter encode error = %v", err)
		}
		if decoded.Player.Name != name {
			t.Fatalf("Player.Name = %q, want %q", decoded.Player.Name, name)
		}
		if decoded.Player.PlayerID != playerID {
			t.Fatalf("PlayerID = %d, want %d", decoded.Player.PlayerID, playerID)
		}
		if decoded.Player.DateCreatedUnix != character.Player.DateCreatedUnix {
			t.Fatalf("DateCreatedUnix = %d, want %d", decoded.Player.DateCreatedUnix, character.Player.DateCreatedUnix)
		}
		if decoded.FileLength != uint32(len(encoded)-fileOverhead) {
			t.Fatalf("FileLength = %d, want %d", decoded.FileLength, len(encoded)-fileOverhead)
		}
		if !decoded.Trailer.HashValid {
			t.Fatal("Trailer.HashValid = false, want true")
		}

		reencoded, err := EncodeBytes(decoded)
		if err != nil {
			t.Fatalf("EncodeBytes after NewCharacter decode error = %v", err)
		}
		if !bytes.Equal(reencoded, encoded) {
			t.Fatalf("NewCharacter encoded data is not stable: reencoded %d bytes, want %d", len(reencoded), len(encoded))
		}
	})
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

func TestCreditCraftedItem(t *testing.T) {
	character := &Character{Player: PlayerData{PlayerID: 123, Name: "Tester"}}

	item := character.CreditCraftedItem(Item{Name: "SwordIron"})
	if item.CrafterID != 123 || item.CrafterName != "Tester" {
		t.Fatalf("crafted item = %+v, want player crafter", item)
	}

	item = character.CreditCraftedItem(Item{Name: "Wood"})
	if item.CrafterID != 0 || item.CrafterName != "" {
		t.Fatalf("non-crafted item = %+v, want no crafter", item)
	}

	item = character.CreditCraftedItem(Item{Name: "SwordIron", CrafterID: 456, CrafterName: "Other"})
	if item.CrafterID != 456 || item.CrafterName != "Other" {
		t.Fatalf("explicit crafter item = %+v, want preserved crafter", item)
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
	character := NewCharacter("Valid Test", 123456)
	character.Player = NewPlayer(character.Player.Name, character.Player.PlayerID)
	character.Trailer.HashValid = true
	return character
}
