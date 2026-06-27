package fch

import "testing"

func TestFoodDecode(t *testing.T) {
	w := NewWriter()
	w.str("CarrotSoup")
	w.f32(693)

	got := readValue[Food, *Food](NewReader(w.Data()))
	want := Food{Name: "CarrotSoup", Time: 693}
	if got != want {
		t.Fatalf("Food = %#v, want %#v", got, want)
	}
}
