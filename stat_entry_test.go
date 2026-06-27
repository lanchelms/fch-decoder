package fch

import "testing"

func TestStatEntryDecode(t *testing.T) {
	w := NewWriter()
	w.str("Kills")
	w.f32(12.5)

	got := readValue[StatEntry, *StatEntry](NewReader(w.Data()))
	want := StatEntry{Name: "Kills", Value: 12.5}
	if got != want {
		t.Fatalf("StatEntry = %#v, want %#v", got, want)
	}
}
