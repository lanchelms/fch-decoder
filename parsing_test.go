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

func TestParseAssignment(t *testing.T) {
	assignment, err := ParseAssignment(" Deaths = 5 ")
	if err != nil {
		t.Fatal(err)
	}
	if assignment.Name != "Deaths" || assignment.Value != 5 {
		t.Fatalf("assignment = %+v", assignment)
	}
}

func TestParseSkillRef(t *testing.T) {
	skill, err := ParseSkillRef("Run")
	if err != nil {
		t.Fatal(err)
	}
	if skill.Type != 102 || skill.Name != "Run" {
		t.Fatalf("ParseSkillRef = %+v", skill)
	}

	skill, err = ParseSkillRef("123")
	if err != nil {
		t.Fatal(err)
	}
	if skill.Type != 123 || skill.Name != "123" {
		t.Fatalf("ParseSkillRef numeric = %+v", skill)
	}
}

func TestParsePlayerStatRef(t *testing.T) {
	stat, err := ParsePlayerStatRef("Builds")
	if err != nil {
		t.Fatal(err)
	}
	if stat.Index != 2 || stat.Name != "Builds" {
		t.Fatalf("ParsePlayerStatRef = %+v", stat)
	}
}
