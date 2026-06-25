package fch

import "testing"

func TestParseInventoryItem(t *testing.T) {
	item, err := ParseInventoryItem("Wood,stack=50,durability=0.75,grid-x=1,grid-y=2,equipped=true,quality=3,variant=4,crafter-id=123,crafter-name=Tester,world-level=2,picked-up=false")
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "Wood" ||
		item.Stack != 50 ||
		item.Durability != 0.75 ||
		item.GridX != 1 ||
		item.GridY != 2 ||
		!item.Equipped ||
		item.Quality != 3 ||
		item.Variant != 4 ||
		item.CrafterID != 123 ||
		item.CrafterName != "Tester" ||
		item.WorldLevel != 2 ||
		item.PickedUp {
		t.Fatalf("item = %+v", item)
	}
}

func TestParseInventoryItemDefaults(t *testing.T) {
	item, err := ParseInventoryItem("Stone")
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "Stone" || item.Stack != 1 || item.Durability != 1 || item.Quality != 1 || !item.PickedUp {
		t.Fatalf("item = %+v, want defaults", item)
	}
}

func TestParseStatAssignment(t *testing.T) {
	assignment, err := ParseStatAssignment(" Deaths = 5 ")
	if err != nil {
		t.Fatal(err)
	}
	if assignment.Name != "Deaths" || assignment.Value != 5 {
		t.Fatalf("assignment = %+v", assignment)
	}
}

func TestParseSkillType(t *testing.T) {
	skillType, name, err := ParseSkillType("Run")
	if err != nil {
		t.Fatal(err)
	}
	if skillType != 102 || name != "Run" {
		t.Fatalf("ParseSkillType = %d, %q", skillType, name)
	}

	skillType, name, err = ParseSkillType("123")
	if err != nil {
		t.Fatal(err)
	}
	if skillType != 123 || name != "123" {
		t.Fatalf("ParseSkillType numeric = %d, %q", skillType, name)
	}
}

func TestParsePlayerStatIndex(t *testing.T) {
	index, name, err := ParsePlayerStatIndex("Builds")
	if err != nil {
		t.Fatal(err)
	}
	if index != 2 || name != "Builds" {
		t.Fatalf("ParsePlayerStatIndex = %d, %q", index, name)
	}
}
