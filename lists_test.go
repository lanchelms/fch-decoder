package fch

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadList(t *testing.T) {
	data := appendU32(nil, 2)
	data = appendU32(data, 10)
	data = appendU32(data, 20)

	got := readList(newReader(data), func(r *reader) uint32 {
		return r.u32()
	})
	want := []uint32{10, 20}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readList = %#v, want %#v", got, want)
	}
}

func TestReadListStrings(t *testing.T) {
	data := appendU32(nil, 2)
	data = appendString(data, "one")
	data = appendString(data, "two")

	got := readList(newReader(data), str)
	want := []string{"one", "two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readList strings = %#v, want %#v", got, want)
	}
}

func TestListReaders(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		read func(*reader) any
		want any
	}{
		{
			name: "statEntry",
			data: appendF32(appendString(nil, "Kills"), 12.5),
			read: func(r *reader) any { return statEntry(r) },
			want: StatEntry{Name: "Kills", Value: 12.5},
		},
		{
			name: "timedEntry",
			data: appendF32(appendString(nil, "World"), 42),
			read: func(r *reader) any { return timedEntry(r) },
			want: TimedEntry{Name: "World", Seconds: 42},
		},
		{
			name: "textEntry",
			data: appendString(appendString(nil, "key"), "value"),
			read: func(r *reader) any { return textEntry(r) },
			want: TextEntry{Key: "key", Value: "value"},
		},
		{
			name: "station",
			data: appendU32(appendString(nil, "Workbench"), 3),
			read: func(r *reader) any { return station(r) },
			want: Station{Name: "Workbench", Level: 3},
		},
		{
			name: "biome",
			data: appendU32(nil, 7),
			read: func(r *reader) any { return biome(r) },
			want: uint32(7),
		},
		{
			name: "food",
			data: appendF32(appendString(nil, "CarrotSoup"), 693),
			read: func(r *reader) any { return food(r) },
			want: Food{Name: "CarrotSoup", Time: 693},
		},
		{
			name: "skill",
			data: appendF32(appendF32(appendU32(nil, 105), 24.75), 0.5),
			read: func(r *reader) any { return skill(r) },
			want: Skill{Type: 105, Name: "Cooking", Level: 24.75, DisplayLevel: 24, Accumulator: 0.5},
		},
		{
			name: "worldKeyRaw",
			data: appendF32(appendString(nil, "nomap"), 7),
			read: func(r *reader) any { return worldKey(r) },
			want: WorldKey{Raw: "nomap", Seconds: 7},
		},
		{
			name: "worldKeySplit",
			data: appendF32(appendString(nil, "PlayerDamage default"), 1375),
			read: func(r *reader) any { return worldKey(r) },
			want: WorldKey{Raw: "PlayerDamage default", Key: "PlayerDamage", Setting: "default", Seconds: 1375},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.read(newReader(tt.data))
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("%s = %#v, want %#v", tt.name, got, tt.want)
			}
		})
	}
}

func TestDecodeConvertedListsFromSample(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "Steam_76561198018104185_bortson.fch"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	got, err := Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	p := got.Player

	if got := p.KnownWorlds[0]; got != (TimedEntry{Name: "LanChelmsDeepNorth2", Seconds: 284707}) {
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
	if got := p.KnownBiomes[0]; got != uint32(1) {
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
