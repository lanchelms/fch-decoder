package fch

import "testing"

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

	character.SetSkillLevel(102, "Run", 22)
	character.SetSkillLevel(102, "Run", 23)
	if len(character.Player.Skills) != 1 || character.Player.Skills[0].Level != 23 {
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
