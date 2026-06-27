package valheim

import "testing"

func TestStationDecode(t *testing.T) {
	w := NewWriter()
	w.String("Workbench")
	w.Uint32(3)

	got := readValue[Station, *Station](NewReader(w.Data()))
	want := Station{Name: "Workbench", Level: 3}
	if got != want {
		t.Fatalf("Station = %#v, want %#v", got, want)
	}
}
