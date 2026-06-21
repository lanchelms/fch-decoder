package fch

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
