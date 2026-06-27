package valheim

import "testing"

func TestSkillDecode(t *testing.T) {
	w := NewWriter()
	w.Uint32(105)
	w.Float32(24.75)
	w.Float32(0.5)

	got := readValue[Skill, *Skill](NewReader(w.Data()))
	want := Skill{Type: 105, Name: "Cooking", Level: 24.75, DisplayLevel: 24, Accumulator: 0.5}
	if got != want {
		t.Fatalf("Skill = %#v, want %#v", got, want)
	}
}
