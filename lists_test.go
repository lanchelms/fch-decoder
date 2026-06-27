package fch

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadList(t *testing.T) {
	w := NewWriter()
	w.u32(2)
	StatEntry{Name: "one", Value: 10}.Encode(w)
	StatEntry{Name: "two", Value: 20}.Encode(w)

	got := readList[StatEntry](NewReader(w.Data()))
	want := []StatEntry{{Name: "one", Value: 10}, {Name: "two", Value: 20}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readList = %#v, want %#v", got, want)
	}
}

func TestReadListStrings(t *testing.T) {
	w := NewWriter()
	w.u32(2)
	w.str("one")
	w.str("two")

	got := readStringList(NewReader(w.Data()))
	want := []string{"one", "two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readList strings = %#v, want %#v", got, want)
	}
}

func TestDecodeConvertedListsFromSample(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "Steam_222222_bortson.fch"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	got, err := Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	p := got.Player

	if got := p.KnownWorlds[0]; got != (TimeEntry{Name: "LanChelmsDeepNorth2", Seconds: 284707}) {
		t.Fatalf("KnownWorlds[0] = %+v", got)
	}
	if got := p.KnownWorldKeys[0]; got != (WorldKey{Raw: "nomap", Seconds: 7}) {
		t.Fatalf("KnownWorldKeys[0] = %+v", got)
	}
	if got := findWorldKey(p.KnownWorldKeys, "PlayerDamage default"); got != (WorldKey{Raw: "PlayerDamage default", Key: "PlayerDamage", Setting: "default", Seconds: 1375}) {
		t.Fatalf("KnownWorldKeys[PlayerDamage default] = %+v", got)
	}
	if got := p.KnownCommands[0]; got != (StatEntry{Name: "say", Value: 6}) {
		t.Fatalf("KnownCommands[0] = %+v", got)
	}
	if got := p.EnemyStats[0]; got != (StatEntry{Name: "$enemy_greyling", Value: 122}) {
		t.Fatalf("EnemyStats[0] = %+v", got)
	}
	if got := p.MaterialStats[0]; got != (StatEntry{Name: "$item_torch", Value: 8}) {
		t.Fatalf("MaterialStats[0] = %+v", got)
	}
	if got := p.RecipeStats[0]; got != (StatEntry{Name: "$item_axe_stone", Value: 1}) {
		t.Fatalf("RecipeStats[0] = %+v", got)
	}
	if got := p.KnownStations[0]; got != (Station{Name: "$piece_workbench", Level: 4}) {
		t.Fatalf("KnownStations[0] = %+v", got)
	}
	if got := p.KnownBiomes[0]; got != Biome(1) {
		t.Fatalf("KnownBiomes[0] = %d", got)
	}
	if got := p.PlayerKnownTexts[0]; got != (TextEntry{Key: "$tutorial_workbench_label", Value: "$tutorial_workbench_text"}) {
		t.Fatalf("PlayerKnownTexts[0] = %+v", got)
	}
	if got := p.Foods[0]; got != (Food{Name: "CarrotSoup", Time: 693}) {
		t.Fatalf("Foods[0] = %+v", got)
	}
	if got := p.Skills[0]; got.Type != 13 || got.Name != "WoodCutting" || got.DisplayLevel != 49 {
		t.Fatalf("Skills[0] = %+v", got)
	}
	if got := p.CustomData[0]; got != (TextEntry{Key: "ACB_PreventPulling", Value: "1"}) {
		t.Fatalf("CustomData[0] = %+v", got)
	}
}

func findWorldKey(keys []WorldKey, raw string) WorldKey {
	for _, key := range keys {
		if key.Raw == raw {
			return key
		}
	}
	return WorldKey{}
}
