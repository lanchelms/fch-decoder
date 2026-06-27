package valheim

import "testing"

func TestSkillName(t *testing.T) {
	tests := []struct {
		name      string
		skillType int32
		want      string
	}{
		{name: "first", skillType: 0, want: "None"},
		{name: "combat", skillType: 8, want: "Bows"},
		{name: "high value", skillType: 999, want: "All"},
		{name: "unknown", skillType: 109, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := skillName(tt.skillType); got != tt.want {
				t.Fatalf("skillName(%d) = %q, want %q", tt.skillType, got, tt.want)
			}
		})
	}
}

func TestSkillTypeByName(t *testing.T) {
	got, ok := SkillTypeByName("swords")
	if !ok || got != 1 {
		t.Fatalf("SkillTypeByName(swords) = %d, %v; want 1, true", got, ok)
	}
	if _, ok := SkillTypeByName("not-a-skill"); ok {
		t.Fatal("SkillTypeByName(not-a-skill) ok = true, want false")
	}
}

func TestSkillNames(t *testing.T) {
	got := SkillNames()
	if len(got) == 0 {
		t.Fatal("SkillNames returned no names")
	}
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Fatalf("SkillNames = %v, want sorted names", got)
		}
	}
}

func TestPlayerStatName(t *testing.T) {
	tests := []struct {
		name  string
		index int
		want  string
	}{
		{name: "first", index: 0, want: "Deaths"},
		{name: "middle", index: 27, want: "TreeChops"},
		{name: "last", index: len(playerStatNames) - 1, want: "UsePowerDeepNorth"},
		{name: "negative", index: -1, want: ""},
		{name: "too high", index: len(playerStatNames), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := playerStatName(tt.index); got != tt.want {
				t.Fatalf("playerStatName(%d) = %q, want %q", tt.index, got, tt.want)
			}
		})
	}
}

func TestPlayerStatIndexByName(t *testing.T) {
	got, ok := PlayerStatIndexByName("deaths")
	if !ok || got != 0 {
		t.Fatalf("PlayerStatIndexByName(deaths) = %d, %v; want 0, true", got, ok)
	}
	if _, ok := PlayerStatIndexByName("not-a-stat"); ok {
		t.Fatal("PlayerStatIndexByName(not-a-stat) ok = true, want false")
	}
}

func TestPlayerStatNames(t *testing.T) {
	got := PlayerStatNames()
	if len(got) == 0 || got[0] != "Deaths" {
		t.Fatalf("PlayerStatNames = %v, want Deaths first", got)
	}
	got[0] = "mutated"
	if PlayerStatNames()[0] != "Deaths" {
		t.Fatal("PlayerStatNames returned mutable package storage")
	}
}
