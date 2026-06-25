package main

import "testing"

func TestParseInventoryItem(t *testing.T) {
	item, positioned, err := parseInventoryItem("Wood,stack=50,durability=0.75,pos=1:2,equipped=true,quality=3,variant=4,crafter-id=123,crafter-name=Tester,world-level=2,picked-up=false")
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
		item.PickedUp ||
		!positioned {
		t.Fatalf("item = %+v positioned = %v", item, positioned)
	}
}

func TestParseInventoryItemDefaults(t *testing.T) {
	item, positioned, err := parseInventoryItem("Stone")
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "Stone" || item.Stack != 1 || item.Durability != 100 || item.Quality != 1 || !item.PickedUp || positioned {
		t.Fatalf("item = %+v positioned = %v, want defaults", item, positioned)
	}
}

func TestParseInventoryItemRejectsUnsafeValues(t *testing.T) {
	tests := []string{
		"",
		"Wood,stack=0",
		"Wood,durability=-1",
		"Wood,durability=NaN",
		"Wood,pos=-1:0",
		"Wood,pos=0:-1",
		"Wood,pos=0",
		"Wood,pos=0:",
		"Wood,pos=:0",
		"Wood,pos=0:0:0",
		"Wood,quality=0",
		"Wood,variant=-1",
	}
	for _, value := range tests {
		if _, _, err := parseInventoryItem(value); err == nil {
			t.Fatalf("parseInventoryItem(%q) error = nil, want error", value)
		}
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

func TestParseInventoryNameRejectsEmpty(t *testing.T) {
	if _, err := parseInventoryName(" "); err == nil {
		t.Fatal("parseInventoryName error = nil, want error")
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

func TestParseSkillRefRejectsNegativeType(t *testing.T) {
	if _, err := parseSkillRef("-1"); err == nil {
		t.Fatal("parseSkillRef error = nil, want error")
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

func TestParsePlayerStatRefRejectsInvalidIndex(t *testing.T) {
	tests := []string{"-1", "2147483647"}
	for _, value := range tests {
		if _, err := parsePlayerStatRef(value); err == nil {
			t.Fatalf("parsePlayerStatRef(%q) error = nil, want error", value)
		}
	}
}

func TestParseSkillLevel(t *testing.T) {
	if _, err := parseSkillLevel(50); err != nil {
		t.Fatal(err)
	}
	for _, value := range []float32{-1, 101} {
		if _, err := parseSkillLevel(value); err == nil {
			t.Fatalf("parseSkillLevel(%v) error = nil, want error", value)
		}
	}
}

func TestParseStatValue(t *testing.T) {
	if _, err := parseStatValue(50); err != nil {
		t.Fatal(err)
	}
	if _, err := parseStatValue(-1); err == nil {
		t.Fatal("parseStatValue(-1) error = nil, want error")
	}
}
