package fch

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/lanchelms/fch-decoder/binary"
	"github.com/lanchelms/fch-decoder/valheim"
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
	if decoded.Player.Inventory[0].CustomData[0] != (valheim.TextEntry{Key: "crafter", Value: "yes"}) {
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
	character := valheim.NewCharacter("New Test", 987654321)
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
	if !decoded.HasPlayerData {
		t.Fatal("HasPlayerData = false, want true")
	}
	if decoded.Player.PlayerVersion != 29 {
		t.Fatalf("PlayerVersion = %d, want 29", decoded.Player.PlayerVersion)
	}
	if decoded.Player.InventoryVersion != 106 {
		t.Fatalf("InventoryVersion = %d, want 106", decoded.Player.InventoryVersion)
	}
	if decoded.Player.SkillVersion != 2 {
		t.Fatalf("SkillVersion = %d, want 2", decoded.Player.SkillVersion)
	}
	if decoded.PlayerDataLength == 0 {
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
		t.Fatalf("Encode Writer bytes differ from EncodeBytes")
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
	character.Player.Inventory = append(character.Player.Inventory, valheim.Item{
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

	if decoded.PlayerDataLength <= character.PlayerDataLength {
		t.Fatalf("PlayerDataLength = %d, want greater than original %d", decoded.PlayerDataLength, character.PlayerDataLength)
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

func syntheticCharacter() *valheim.Character {
	return &valheim.Character{
		Version:         43,
		PlayerStatCount: 2,
		PlayerStats: []valheim.StatEntry{
			{Name: "Deaths", Value: 1},
			{Name: "CraftingStationUses", Value: 2.5},
		},
		Map:              valheim.Map{Raw: syntheticMapSection()},
		HasPlayerData:    true,
		PlayerDataLength: 999,
		Player: valheim.Player{
			Name:            "Encoder Test",
			PlayerID:        123456,
			StartSeed:       "seed",
			UsedCheats:      true,
			DateCreatedUnix: 1780027200,
			KnownWorlds: []valheim.TimeEntry{
				{Name: "world", Seconds: 12.5},
			},
			KnownWorldKeys: []valheim.WorldKey{
				{Key: "defeated_eikthyr", Setting: "True", Seconds: 9},
			},
			KnownCommands: []valheim.StatEntry{
				{Name: "god", Value: 1},
			},
			EnemyStats: []valheim.StatEntry{
				{Name: "Greydwarf", Value: 3},
			},
			MaterialStats: []valheim.StatEntry{
				{Name: "Wood", Value: 4},
			},
			RecipeStats: []valheim.StatEntry{
				{Name: "Hammer", Value: 5},
			},
			PlayerState: valheim.PlayerState{
				PlayerVersion:  29,
				MaxHealth:      100,
				Health:         75,
				MaxStamina:     120,
				TimeSinceDeath: 6,
				GuardianPower: valheim.GuardianPower{
					Name:     "GP_Eikthyr",
					Cooldown: 7,
				},
				InventoryVersion: 106,
				Inventory: []valheim.Item{
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
						CustomData: []valheim.TextEntry{
							{Key: "crafter", Value: "yes"},
						},
						WorldLevel: 7,
						PickedUp:   true,
					},
				},
			},
			PlayerTail: valheim.PlayerTail{
				KnownRecipes: []string{"Hammer"},
				KnownStations: []valheim.Station{
					{Name: "piece_workbench", Level: 2},
				},
				KnownMaterials: []string{"Wood"},
				ShownTutorials: []string{"tutorial"},
				Uniques:        []string{"unique"},
				Trophies:       []string{"TrophyDeer"},
				KnownBiomes:    []valheim.Biome{1},
				PlayerKnownTexts: []valheim.TextEntry{
					{Key: "raven", Value: "text"},
				},
				Beard:        "Beard1",
				Hair:         "Hair1",
				SkinColor:    valheim.Vector3{X: 0.1, Y: 0.2, Z: 0.3},
				HairColor:    valheim.Vector3{X: 0.4, Y: 0.5, Z: 0.6},
				ModelIndex:   1,
				SkillVersion: 2,
				Foods: []valheim.Food{
					{Name: "CookedMeat", Time: 10},
				},
				Skills: []valheim.Skill{
					{Type: 1, Level: 12.75, Accumulator: 0.5},
				},
				CustomData: []valheim.TextEntry{
					{Key: "custom", Value: "data"},
				},
				Stamina: 84,
				MaxEitr: 50,
				Eitr:    25,
			},
		},
	}
}

func syntheticMapSection() []byte {
	w := binary.NewWriter()
	w.Uint32(3)
	w.Uint32(0)
	w.Uint32(3)
	raw := w.Data()
	return append(raw, 0x1f, 0x8b, 0x08)
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) {
	return len(p) - 1, nil
}
