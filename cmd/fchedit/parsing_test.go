package main

import "testing"

func TestParseInventoryItem(t *testing.T) {
	item, err := parseInventoryItem("Wood,stack=50,durability=0.75,grid-x=1,grid-y=2,equipped=true,quality=3,variant=4,crafter-id=123,crafter-name=Tester,world-level=2,picked-up=false")
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
	item, err := parseInventoryItem("Stone")
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "Stone" || item.Stack != 1 || item.Durability != 1 || item.Quality != 1 || !item.PickedUp {
		t.Fatalf("item = %+v, want defaults", item)
	}
}

func TestParseInventoryAction(t *testing.T) {
	item, err := parseInventoryAction(addInventory, "Stone,stack=10")
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "Stone" || item.Stack != 10 {
		t.Fatalf("add item = %+v", item)
	}

	item, err = parseInventoryAction(removeInventory, " Wood ")
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "Wood" {
		t.Fatalf("remove item = %+v", item)
	}
}

func TestParseAssignment(t *testing.T) {
	assignment, err := parseAssignment(" Deaths = 5 ")
	if err != nil {
		t.Fatal(err)
	}
	if assignment.name != "Deaths" || assignment.value != 5 {
		t.Fatalf("assignment = %+v", assignment)
	}
}

func TestParseSkillRef(t *testing.T) {
	skill, err := parseSkillRef("Run")
	if err != nil {
		t.Fatal(err)
	}
	if skill.skillType != 102 || skill.name != "Run" {
		t.Fatalf("parseSkillRef = %+v", skill)
	}

	skill, err = parseSkillRef("123")
	if err != nil {
		t.Fatal(err)
	}
	if skill.skillType != 123 || skill.name != "123" {
		t.Fatalf("parseSkillRef numeric = %+v", skill)
	}
}

func TestParsePlayerStatRef(t *testing.T) {
	stat, err := parsePlayerStatRef("Builds")
	if err != nil {
		t.Fatal(err)
	}
	if stat.index != 2 || stat.name != "Builds" {
		t.Fatalf("parsePlayerStatRef = %+v", stat)
	}
}
