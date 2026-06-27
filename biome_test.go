package fch

import "testing"

func TestBiomeDecode(t *testing.T) {
	w := NewWriter()
	w.u32(7)

	got := readValue[Biome, *Biome](NewReader(w.Data()))
	want := Biome(7)
	if got != want {
		t.Fatalf("Biome = %#v, want %#v", got, want)
	}
}
