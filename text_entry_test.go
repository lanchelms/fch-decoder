package fch

import "testing"

func TestTextEntryDecode(t *testing.T) {
	w := NewWriter()
	w.str("key")
	w.str("value")

	got := readValue[TextEntry, *TextEntry](NewReader(w.Data()))
	want := TextEntry{Key: "key", Value: "value"}
	if got != want {
		t.Fatalf("TextEntry = %#v, want %#v", got, want)
	}
}
