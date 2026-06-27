package valheim

import "testing"

func TestStatEntryDecode(t *testing.T) {
	w := NewWriter()
	w.String("Kills")
	w.Float32(12.5)

	got := readValue[StatEntry, *StatEntry](NewReader(w.Data()))
	want := StatEntry{Name: "Kills", Value: 12.5}
	if got != want {
		t.Fatalf("StatEntry = %#v, want %#v", got, want)
	}
}
