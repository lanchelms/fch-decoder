package fch

import "testing"

func TestItemDecodeCustomData(t *testing.T) {
	w := NewWriter()
	writeList(w, []Item{{
		Name:        "Hammer",
		Stack:       2,
		Durability:  3.5,
		GridX:       4,
		GridY:       5,
		Equipped:    true,
		Quality:     6,
		Variant:     7,
		CrafterID:   8,
		CrafterName: "Crafter",
		CustomData: []TextEntry{{
			Key:   "custom-key",
			Value: "custom-value",
		}},
		WorldLevel: 9,
		PickedUp:   true,
	}})

	got := readList[Item](NewReader(w.Data()))
	if len(got) != 1 {
		t.Fatalf("Inventory = %d, want 1", len(got))
	}
	if got := got[0].CustomData; len(got) != 1 || got[0] != (TextEntry{Key: "custom-key", Value: "custom-value"}) {
		t.Fatalf("Inventory[0].CustomData = %+v", got)
	}
}
