package fch

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestEncodeFixtureRoundTripsByteForByte(t *testing.T) {
	files := []string{
		"Steam_111111_fenris bueller.fch",
		"Steam_222222_bortson.fch",
		"Steam_333333_tugen.fch",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			original, err := os.ReadFile(filepath.Join("testdata", file))
			if err != nil {
				t.Fatal(err)
			}
			character, err := DecodeBytes(original)
			if err != nil {
				t.Fatal(err)
			}

			encoded, err := EncodeBytes(character)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(encoded, original) {
				t.Fatalf("encoded fixture differs from original: got %d bytes, want %d", len(encoded), len(original))
			}
			roundTrip, err := DecodeBytes(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if !roundTrip.Trailer.HashValid {
				t.Fatal("round-trip trailer hash is invalid")
			}
		})
	}
}

func TestEncodeSyntheticCharacterDecodes(t *testing.T) {
	character := syntheticCharacter()

	encoded, err := EncodeBytes(character)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeBytes(encoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.FileLength != uint32(len(encoded)-fileOverhead) {
		t.Fatalf("FileLength = %d, want %d", decoded.FileLength, len(encoded)-fileOverhead)
	}
	if decoded.Version != character.Version {
		t.Fatalf("Version = %d, want %d", decoded.Version, character.Version)
	}
	if decoded.PlayerStatCount != uint32(len(character.PlayerStats)) {
		t.Fatalf("PlayerStatCount = %d, want %d", decoded.PlayerStatCount, len(character.PlayerStats))
	}
	if got := decoded.PlayerStats[0].Value; got != character.PlayerStats[0].Value {
		t.Fatalf("PlayerStats[0].Value = %v, want %v", got, character.PlayerStats[0].Value)
	}
	if decoded.Map.StoredLength != 3 || decoded.Map.CompressedLength != 3 {
		t.Fatalf("bad map metadata: %+v", decoded.Map)
	}
	if decoded.Player.Name != character.Player.Name {
		t.Fatalf("Player.Name = %q, want %q", decoded.Player.Name, character.Player.Name)
	}
	if decoded.Player.KnownWorldKeys[0].Raw != "defeated_eikthyr True" {
		t.Fatalf("KnownWorldKeys[0].Raw = %q", decoded.Player.KnownWorldKeys[0].Raw)
	}
	if decoded.Player.Inventory[0].CustomData[0] != (TextEntry{Key: "crafter", Value: "yes"}) {
		t.Fatalf("Inventory[0].CustomData = %+v", decoded.Player.Inventory[0].CustomData)
	}
	if decoded.Player.Skills[0].DisplayLevel != 12 {
		t.Fatalf("Skills[0].DisplayLevel = %d, want 12", decoded.Player.Skills[0].DisplayLevel)
	}
	if decoded.Player.Stamina != 84 || decoded.Player.MaxEitr != 50 || decoded.Player.Eitr != 25 {
		t.Fatalf("tail floats = stamina %v maxEitr %v eitr %v, want 84/50/25", decoded.Player.Stamina, decoded.Player.MaxEitr, decoded.Player.Eitr)
	}
	if !decoded.Trailer.HashValid {
		t.Fatal("Trailer.HashValid = false, want true")
	}
	if decoded.RemainingBytes != 0 {
		t.Fatalf("RemainingBytes = %d, want 0", decoded.RemainingBytes)
	}
}

func TestEncodeNewCharacterWithPlayerData(t *testing.T) {
	character := NewCharacter("New Test", 987654321)
	encoded, err := EncodeBytes(character)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeBytes(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Player.Name != "New Test" {
		t.Fatalf("Player.Name = %q, want New Test", decoded.Player.Name)
	}
	if decoded.Player.PlayerID != 987654321 {
		t.Fatalf("PlayerID = %d, want 987654321", decoded.Player.PlayerID)
	}
	if decoded.Player.DateCreatedUnix != character.Player.DateCreatedUnix {
		t.Fatalf("DateCreatedUnix = %d, want %d", decoded.Player.DateCreatedUnix, character.Player.DateCreatedUnix)
	}
	if !decoded.Player.HasPlayerData {
		t.Fatal("HasPlayerData = false, want true")
	}
	if decoded.Player.PlayerVersion != supportedPlayerVersion {
		t.Fatalf("PlayerVersion = %d, want %d", decoded.Player.PlayerVersion, supportedPlayerVersion)
	}
	if decoded.Player.InventoryVersion != supportedInventoryVersion {
		t.Fatalf("InventoryVersion = %d, want %d", decoded.Player.InventoryVersion, supportedInventoryVersion)
	}
	if decoded.Player.SkillVersion != supportedSkillVersion {
		t.Fatalf("SkillVersion = %d, want %d", decoded.Player.SkillVersion, supportedSkillVersion)
	}
	if decoded.Player.PlayerDataLength == 0 {
		t.Fatal("PlayerDataLength = 0, want encoded player data")
	}
	if !decoded.Trailer.HashValid {
		t.Fatal("Trailer.HashValid = false, want true")
	}
	if decoded.RemainingBytes != 0 {
		t.Fatalf("RemainingBytes = %d, want 0", decoded.RemainingBytes)
	}
}

func TestEncodeWritesToWriter(t *testing.T) {
	character := syntheticCharacter()
	want, err := EncodeBytes(character)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := Encode(&buf, character); err != nil {
		t.Fatal(err)
	}
	if got := buf.Bytes(); !bytes.Equal(got, want) {
		t.Fatalf("Encode writer bytes differ from EncodeBytes")
	}
}

func TestEncodeDetectsShortWrite(t *testing.T) {
	err := Encode(shortWriter{}, syntheticCharacter())
	if err != io.ErrShortWrite {
		t.Fatalf("Encode error = %v, want short write", err)
	}
}

func TestEncodeRehashesMutatedPayload(t *testing.T) {
	original, err := os.ReadFile(filepath.Join("testdata", "Steam_333333_tugen.fch"))
	if err != nil {
		t.Fatal(err)
	}
	character, err := DecodeBytes(original)
	if err != nil {
		t.Fatal(err)
	}
	oldHash := append([]byte(nil), character.Trailer.Hash...)
	character.PlayerStats[0].Value++

	encoded, err := EncodeBytes(character)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeBytes(encoded)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(decoded.Trailer.Hash, oldHash) {
		t.Fatal("Trailer.Hash was not recalculated")
	}
	if !decoded.Trailer.HashValid {
		t.Fatal("Trailer.HashValid = false, want true")
	}
	if got := decoded.PlayerStats[0].Value; got != character.PlayerStats[0].Value {
		t.Fatalf("mutated PlayerStats[0].Value = %v, want %v", got, character.PlayerStats[0].Value)
	}
}

func TestEncodeRecalculatesPlayerDataLength(t *testing.T) {
	original, err := os.ReadFile(filepath.Join("testdata", "Steam_333333_tugen.fch"))
	if err != nil {
		t.Fatal(err)
	}
	character, err := DecodeBytes(original)
	if err != nil {
		t.Fatal(err)
	}
	character.Player.Inventory = append(character.Player.Inventory, Item{
		Name:        "Wood",
		Stack:       50,
		Durability:  1,
		Quality:     1,
		CrafterName: "fchedit",
		PickedUp:    true,
	})

	encoded, err := EncodeBytes(character)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeBytes(encoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Player.PlayerDataLength <= character.Player.PlayerDataLength {
		t.Fatalf("PlayerDataLength = %d, want greater than original %d", decoded.Player.PlayerDataLength, character.Player.PlayerDataLength)
	}
	if len(decoded.Player.Inventory) != len(character.Player.Inventory) {
		t.Fatalf("Inventory = %d, want %d", len(decoded.Player.Inventory), len(character.Player.Inventory))
	}
	if !decoded.Trailer.HashValid {
		t.Fatal("Trailer.HashValid = false, want true")
	}
}

func TestEncodeRejectsNilCharacter(t *testing.T) {
	_, err := EncodeBytes(nil)
	if err == nil || err.Error() != "fch: cannot encode nil character" {
		t.Fatalf("EncodeBytes(nil) error = %v, want nil character", err)
	}
}

func TestEncodeRejectsMissingRawMapSection(t *testing.T) {
	character := syntheticCharacter()
	character.Map.Raw = nil

	_, err := EncodeBytes(character)
	if err == nil || err.Error() != "fch: cannot encode character without raw map section" {
		t.Fatalf("EncodeBytes error = %v, want missing raw map section", err)
	}
}

func TestEncodeRejectsPlayerStatCountMismatch(t *testing.T) {
	character := syntheticCharacter()
	character.PlayerStatCount = uint32(len(character.PlayerStats) + 1)

	_, err := EncodeBytes(character)
	if err == nil || err.Error() != "fch: player stat count 3 does not match 2 stats" {
		t.Fatalf("EncodeBytes error = %v, want player stat count mismatch", err)
	}
}

func TestEncodeRejectsUnsupportedTailFloatCount(t *testing.T) {
	character := syntheticCharacter()
	character.Player.tailFloatCount = 1

	_, err := EncodeBytes(character)
	if err == nil || err.Error() != "fch: unsupported player tail float count 1" {
		t.Fatalf("EncodeBytes error = %v, want unsupported tail float count", err)
	}
}

func syntheticCharacter() *Character {
	return &Character{
		Version:         43,
		PlayerStatCount: 2,
		PlayerStats: []StatEntry{
			{Name: "Deaths", Value: 1},
			{Name: "CraftingStationUses", Value: 2.5},
		},
		Map: MapSection{Raw: syntheticMapSection()},
		Player: Player{
			Name:            "Encoder Test",
			PlayerID:        123456,
			StartSeed:       "seed",
			UsedCheats:      true,
			DateCreatedUnix: 1780027200,
			KnownWorlds: []TimedEntry{
				{Name: "world", Seconds: 12.5},
			},
			KnownWorldKeys: []WorldKey{
				{Key: "defeated_eikthyr", Setting: "True", Seconds: 9},
			},
			KnownCommands: []StatEntry{
				{Name: "god", Value: 1},
			},
			EnemyStats: []StatEntry{
				{Name: "Greydwarf", Value: 3},
			},
			MaterialStats: []StatEntry{
				{Name: "Wood", Value: 4},
			},
			RecipeStats: []StatEntry{
				{Name: "Hammer", Value: 5},
			},
			HasPlayerData:    true,
			PlayerDataLength: 999,
			PlayerVersion:    29,
			MaxHealth:        100,
			Health:           75,
			MaxStamina:       120,
			Stamina:          84,
			MaxEitr:          50,
			Eitr:             25,
			TimeSinceDeath:   6,
			GuardianPower: GuardianPower{
				Name:     "GP_Eikthyr",
				Cooldown: 7,
			},
			InventoryVersion: 106,
			Inventory: []Item{
				{
					Name:        "Hammer",
					Stack:       1,
					Durability:  99,
					GridX:       2,
					GridY:       3,
					Equipped:    true,
					Quality:     4,
					Variant:     5,
					CrafterID:   6,
					CrafterName: "Crafter",
					CustomData: []TextEntry{
						{Key: "crafter", Value: "yes"},
					},
					WorldLevel: 7,
					PickedUp:   true,
				},
			},
			KnownRecipes: []string{"Hammer"},
			KnownStations: []Station{
				{Name: "piece_workbench", Level: 2},
			},
			KnownMaterials: []string{"Wood"},
			ShownTutorials: []string{"tutorial"},
			Uniques:        []string{"unique"},
			Trophies:       []string{"TrophyDeer"},
			KnownBiomes:    []uint32{1},
			PlayerKnownTexts: []TextEntry{
				{Key: "raven", Value: "text"},
			},
			Beard:      "Beard1",
			Hair:       "Hair1",
			SkinColor:  Vector3{X: 0.1, Y: 0.2, Z: 0.3},
			HairColor:  Vector3{X: 0.4, Y: 0.5, Z: 0.6},
			ModelIndex: 1,
			Foods: []Food{
				{Name: "CookedMeat", Time: 10},
			},
			SkillVersion: 2,
			Skills: []Skill{
				{Type: 1, Level: 12.75, Accumulator: 0.5},
			},
			CustomData: []TextEntry{
				{Key: "custom", Value: "data"},
			},
		},
	}
}

func syntheticMapSection() []byte {
	w := newWriter()
	w.u32(3)
	w.u32(0)
	w.u32(3)
	raw := w.data()
	return append(raw, 0x1f, 0x8b, 0x08)
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) {
	return len(p) - 1, nil
}
