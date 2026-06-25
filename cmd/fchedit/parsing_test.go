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

func TestParseInventoryName(t *testing.T) {
	name, err := parseInventoryName(" Wood ")
	if err != nil {
		t.Fatal(err)
	}
	if name != "Wood" {
		t.Fatalf("parseInventoryName = %q, want Wood", name)
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
