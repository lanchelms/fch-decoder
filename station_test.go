package fch

import "testing"

func TestStationDecode(t *testing.T) {
	w := NewWriter()
	w.str("Workbench")
	w.u32(3)

	got := readValue[Station, *Station](NewReader(w.Data()))
	want := Station{Name: "Workbench", Level: 3}
	if got != want {
		t.Fatalf("Station = %#v, want %#v", got, want)
	}
}
