package fch

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeSamples(t *testing.T) {
	tests := []struct {
		file            string
		name            string
		playerID        uint64
		inventory       int
		materials       int
		recipes         int
		knownMaterials  int
		knownRecipes    int
		uniques         int
		trophies        int
		beard           string
		hair            string
		modelIndex      uint32
		dateCreatedUnix int64
		playerLength    uint32
	}{
		{
			file:            "Steam_111111_fenris bueller.fch",
			name:            "Fenris Bueller",
			playerID:        111111,
			inventory:       16,
			materials:       108,
			recipes:         35,
			knownMaterials:  117,
			knownRecipes:    189,
			uniques:         4,
			trophies:        10,
			beard:           "Beard23",
			hair:            "Hair33",
			modelIndex:      0,
			dateCreatedUnix: 1780027200,
			playerLength:    7593,
		},
		{
			file:            "Steam_222222_bortson.fch",
			name:            "Bortson",
			playerID:        222222,
			inventory:       16,
			materials:       141,
			recipes:         54,
			knownMaterials:  143,
			knownRecipes:    197,
			uniques:         4,
			trophies:        13,
			beard:           "",
			hair:            "Hair18",
			modelIndex:      0,
			dateCreatedUnix: 1780113600,
			playerLength:    10122,
		},
		{
			file:            "Steam_333333_tugen.fch",
			name:            "Tugen",
			playerID:        333333,
			inventory:       23,
			materials:       126,
			recipes:         43,
			knownMaterials:  130,
			knownRecipes:    226,
			uniques:         3,
			trophies:        11,
			beard:           "",
			hair:            "",
			modelIndex:      0,
			dateCreatedUnix: 1780027200,
			playerLength:    10262,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(filepath.Join("testdata", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			got, err := Decode(f)
			if err != nil {
				t.Fatal(err)
			}
			if got.Version != 43 {
				t.Fatalf("Version = %d, want 43", got.Version)
			}
			if got.PlayerStatCount != 105 {
				t.Fatalf("PlayerStatCount = %d, want 105", got.PlayerStatCount)
			}
			if len(got.PlayerStats) != int(got.PlayerStatCount) {
				t.Fatalf("PlayerStats = %d, want %d", len(got.PlayerStats), got.PlayerStatCount)
			}
			if got.PlayerStats[0].Name != "Deaths" || got.PlayerStats[104].Name != "UsePowerDeepNorth" {
				t.Fatalf("bad player stat names: first=%q last=%q", got.PlayerStats[0].Name, got.PlayerStats[104].Name)
			}
			if got.Player.Name != tt.name {
				t.Fatalf("Name = %q, want %q", got.Player.Name, tt.name)
			}
			if got.Player.PlayerID != tt.playerID {
				t.Fatalf("PlayerID = %d, want %d", got.Player.PlayerID, tt.playerID)
			}
			if got.Player.StartSeed != "" {
				t.Fatalf("StartSeed = %q, want empty", got.Player.StartSeed)
			}
			if got.Player.DateCreatedUnix != tt.dateCreatedUnix {
				t.Fatalf("DateCreatedUnix = %d, want %d", got.Player.DateCreatedUnix, tt.dateCreatedUnix)
			}
			if got.Player.UsedCheats {
				t.Fatal("UsedCheats = true, want false")
			}
			if len(got.Player.KnownWorlds) != 1 {
				t.Fatalf("KnownWorlds = %d, want 1", len(got.Player.KnownWorlds))
			}
			if got.Player.KnownWorlds[0].Name != "LanChelmsDeepNorth2" {
				t.Fatalf("KnownWorld name = %q", got.Player.KnownWorlds[0].Name)
			}
			if got.Player.KnownWorlds[0].Seconds <= 0 {
				t.Fatalf("KnownWorld seconds = %v, want positive", got.Player.KnownWorlds[0].Seconds)
			}
			if len(got.Player.KnownWorldKeys) == 0 {
				t.Fatal("KnownWorldKeys is empty")
			}
			if got.Player.KnownWorldKeys[0].Raw == "" || got.Player.KnownWorldKeys[0].Seconds <= 0 {
				t.Fatalf("bad first KnownWorldKey: %+v", got.Player.KnownWorldKeys[0])
			}
			if got.Player.GuardianPower.Name != "GP_Eikthyr" {
				t.Fatalf("GuardianPower = %q", got.Player.GuardianPower.Name)
			}
			if !got.HasPlayerData {
				t.Fatal("HasPlayerData = false, want true")
			}
			if got.PlayerDataLength != tt.playerLength {
				t.Fatalf("PlayerDataLength = %d, want %d", got.PlayerDataLength, tt.playerLength)
			}
			if got.Player.PlayerVersion != 29 {
				t.Fatalf("PlayerVersion = %d, want 29", got.Player.PlayerVersion)
			}
			if got.Player.Health <= 0 || got.Player.MaxHealth <= 0 || got.Player.MaxStamina <= 0 {
				t.Fatalf("bad player vitals: health=%v maxHealth=%v maxStamina=%v", got.Player.Health, got.Player.MaxHealth, got.Player.MaxStamina)
			}
			if got.Player.InventoryVersion != 106 {
				t.Fatalf("InventoryVersion = %d, want 106", got.Player.InventoryVersion)
			}
			if len(got.Player.Inventory) != tt.inventory {
				t.Fatalf("Inventory = %d, want %d", len(got.Player.Inventory), tt.inventory)
			}
			if got.Player.Inventory[0].Name == "" {
				t.Fatal("first inventory item has empty name")
			}
			if got.Player.Inventory[0].WorldLevel != 0 {
				t.Fatalf("first inventory item worldLevel = %d, want 0", got.Player.Inventory[0].WorldLevel)
			}
			if got.Player.SkillVersion != 2 {
				t.Fatalf("SkillVersion = %d, want 2", got.Player.SkillVersion)
			}
			if len(got.Player.Skills) != 24 {
				t.Fatalf("Skills = %d, want 24", len(got.Player.Skills))
			}
			if got.Player.Skills[0].Name == "" {
				t.Fatalf("first skill has no mapped name: type=%d", got.Player.Skills[0].Type)
			}
			wantDisplayLevel := int32(math.Floor(float64(got.Player.Skills[0].Level)))
			if got.Player.Skills[0].DisplayLevel != wantDisplayLevel {
				t.Fatalf("first skill display level = %d, want %d", got.Player.Skills[0].DisplayLevel, wantDisplayLevel)
			}
			wantSkillNames := map[int32]string{
				105: "Cooking",
				106: "Farming",
				107: "Crafting",
				108: "Dodge",
			}
			for skillType, wantName := range wantSkillNames {
				gotName := ""
				for _, skill := range got.Player.Skills {
					if skill.Type == skillType {
						gotName = skill.Name
						break
					}
				}
				if gotName != wantName {
					t.Fatalf("skill %d name = %q, want %q", skillType, gotName, wantName)
				}
			}
			inventoryJSON, err := json.Marshal(got.Player.Inventory[0])
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(inventoryJSON), "unknownTail") || strings.Contains(string(inventoryJSON), "unknownByte") {
				t.Fatalf("inventory item JSON includes placeholder fields: %s", inventoryJSON)
			}
			characterJSON, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(characterJSON), "knownCommands") || strings.Contains(string(characterJSON), "shownTutorials") || strings.Contains(string(characterJSON), "playerKnownTexts") {
				t.Fatalf("character JSON includes skipped player fields: %s", characterJSON)
			}
			if len(got.Player.MaterialStats) != tt.materials {
				t.Fatalf("MaterialStats = %d, want %d", len(got.Player.MaterialStats), tt.materials)
			}
			if len(got.Player.RecipeStats) != tt.recipes {
				t.Fatalf("RecipeStats = %d, want %d", len(got.Player.RecipeStats), tt.recipes)
			}
			if len(got.Player.KnownMaterials) != tt.knownMaterials {
				t.Fatalf("KnownMaterials = %d, want %d", len(got.Player.KnownMaterials), tt.knownMaterials)
			}
			if len(got.Player.KnownRecipes) != tt.knownRecipes {
				t.Fatalf("KnownRecipes = %d, want %d", len(got.Player.KnownRecipes), tt.knownRecipes)
			}
			if len(got.Player.Uniques) != tt.uniques {
				t.Fatalf("Uniques = %d, want %d", len(got.Player.Uniques), tt.uniques)
			}
			if len(got.Player.Trophies) != tt.trophies {
				t.Fatalf("Trophies = %d, want %d", len(got.Player.Trophies), tt.trophies)
			}
			if got.Player.Beard != tt.beard {
				t.Fatalf("Beard = %q, want %q", got.Player.Beard, tt.beard)
			}
			if got.Player.Hair != tt.hair {
				t.Fatalf("Hair = %q, want %q", got.Player.Hair, tt.hair)
			}
			if got.Player.ModelIndex != tt.modelIndex {
				t.Fatalf("ModelIndex = %d, want %d", got.Player.ModelIndex, tt.modelIndex)
			}
			if got.Map.Offset == 0 || got.Map.CompressedLength == 0 || got.Map.StoredLength == 0 {
				t.Fatalf("bad map metadata: offset=%d compressedLength=%d storedLength=%d", got.Map.Offset, got.Map.CompressedLength, got.Map.StoredLength)
			}
			if got.Trailer.Length != 64 || len(got.Trailer.Hash) != 64 {
				t.Fatalf("bad trailer: length=%d hash=%d", got.Trailer.Length, len(got.Trailer.Hash))
			}
			if !got.Trailer.HashValid {
				t.Fatal("Trailer.HashValid = false, want true")
			}
			if got.RemainingBytes != 0 {
				t.Fatalf("RemainingBytes = %d, want 0", got.RemainingBytes)
			}
		})
	}
}

func TestDecodeDetectsInvalidTrailerHash(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "Steam_333333_tugen.fch"))
	if err != nil {
		t.Fatal(err)
	}
	data = append([]byte(nil), data...)
	data[12] ^= 1

	got, err := DecodeBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Trailer.HashValid {
		t.Fatal("Trailer.HashValid = true, want false")
	}
}

func TestDecodeMinimalCharacterFixtureWithoutPlayer(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "decoder", "minimal_no_player_data.fch"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeBytes(fixture)
	if err != nil {
		t.Fatal(err)
	}

	assertMinimalCharacter(t, got, "Minimal Test", 987654321, 1780027200)
}

func TestDecodeMalformedInputReturnsError(t *testing.T) {
	data := make([]byte, fileOverhead+16)
	binary.LittleEndian.PutUint32(data[:4], uint32(len(data)-fileOverhead))
	binary.LittleEndian.PutUint32(data[8:12], math.MaxUint32)

	if _, err := DecodeBytes(data); err == nil {
		t.Fatal("DecodeBytes error = nil, want malformed input error")
	}
}

func FuzzDecodeEncode(f *testing.F) {
	files := []string{
		"Steam_111111_fenris bueller.fch",
		"Steam_222222_bortson.fch",
		"Steam_333333_tugen.fch",
	}
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Add([]byte{})
	f.Add([]byte{0, 1, 2, 3, 4})

	f.Fuzz(func(t *testing.T, data []byte) {
		character, err := DecodeBytes(data)
		if err != nil {
			return
		}
		encoded, err := EncodeBytes(character)
		if err != nil {
			t.Fatalf("EncodeBytes after successful decode error = %v", err)
		}
		decoded, err := DecodeBytes(encoded)
		if err != nil {
			t.Fatalf("DecodeBytes after encode error = %v", err)
		}
		if !decoded.Trailer.HashValid {
			t.Fatal("encoded trailer hash is invalid")
		}
	})
}

func TestReadInventoryCustomData(t *testing.T) {
	encoded := newEncoder()
	encoded.inventory([]Item{{
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

	got := readInventory(newReader(encoded.w.data()))
	if len(got) != 1 {
		t.Fatalf("Inventory = %d, want 1", len(got))
	}
	if got := got[0].CustomData; len(got) != 1 || got[0] != (TextEntry{Key: "custom-key", Value: "custom-value"}) {
		t.Fatalf("Inventory[0].CustomData = %+v", got)
	}
}

func minimalMapSection() []byte {
	return []byte{1, 0, 0, 0, 0}
}

func assertMinimalCharacter(t *testing.T, got *Character, name string, playerID uint64, dateCreatedUnix int64) {
	t.Helper()

	if got.Version != 43 {
		t.Fatalf("Version = %d, want 43", got.Version)
	}
	if got.PlayerStatCount != 105 {
		t.Fatalf("PlayerStatCount = %d, want 105", got.PlayerStatCount)
	}
	if len(got.PlayerStats) != 105 {
		t.Fatalf("PlayerStats = %d, want 105", len(got.PlayerStats))
	}
	if got.PlayerStats[0].Name != "Deaths" || got.PlayerStats[104].Name != "UsePowerDeepNorth" {
		t.Fatalf("bad player stat names: first=%q last=%q", got.PlayerStats[0].Name, got.PlayerStats[104].Name)
	}
	if got.Map.CompressedLength != 0 || got.Map.StoredLength != 0 {
		t.Fatalf("bad map lengths: compressed=%d stored=%d", got.Map.CompressedLength, got.Map.StoredLength)
	}
	if string(got.Map.Raw) != string(minimalMapSection()) {
		t.Fatalf("Map.Raw = %v, want %v", got.Map.Raw, minimalMapSection())
	}
	if got.Player.Name != name {
		t.Fatalf("Name = %q, want %q", got.Player.Name, name)
	}
	if got.Player.PlayerID != playerID {
		t.Fatalf("PlayerID = %d, want %d", got.Player.PlayerID, playerID)
	}
	if got.Player.StartSeed != "" {
		t.Fatalf("StartSeed = %q, want empty", got.Player.StartSeed)
	}
	if got.Player.UsedCheats {
		t.Fatal("UsedCheats = true, want false")
	}
	if got.Player.DateCreatedUnix != dateCreatedUnix {
		t.Fatalf("DateCreatedUnix = %d, want %d", got.Player.DateCreatedUnix, dateCreatedUnix)
	}
	if !emptyMinimalLists(got.Player) {
		t.Fatalf("minimal player lists are not empty: %+v", got.Player)
	}
	if got.HasPlayerData {
		t.Fatal("HasPlayerData = true, want false")
	}
	if got.PlayerDataLength != 0 {
		t.Fatalf("PlayerDataLength = %d, want 0", got.PlayerDataLength)
	}
	if got.Player.PlayerVersion != 0 || got.Player.InventoryVersion != 0 || got.Player.SkillVersion != 0 {
		t.Fatalf("player versions = %d/%d/%d, want zero", got.Player.PlayerVersion, got.Player.InventoryVersion, got.Player.SkillVersion)
	}
	if got.Player.Health != 0 || got.Player.MaxHealth != 0 || got.Player.MaxStamina != 0 || got.Player.TimeSinceDeath != 0 {
		t.Fatalf("player vitals are not zero: %+v", got.Player)
	}
	if got.Player.GuardianPower != (GuardianPower{}) {
		t.Fatalf("GuardianPower = %+v, want zero", got.Player.GuardianPower)
	}
	if got.Trailer.Length != 64 || len(got.Trailer.Hash) != 64 {
		t.Fatalf("bad trailer: length=%d hash=%d", got.Trailer.Length, len(got.Trailer.Hash))
	}
	if !got.Trailer.HashValid {
		t.Fatal("Trailer.HashValid = false, want true")
	}
	if got.RemainingBytes != 0 {
		t.Fatalf("RemainingBytes = %d, want 0", got.RemainingBytes)
	}
}

func emptyMinimalLists(p Player) bool {
	return len(p.KnownWorlds) == 0 &&
		len(p.KnownWorldKeys) == 0 &&
		len(p.KnownCommands) == 0 &&
		len(p.EnemyStats) == 0 &&
		len(p.MaterialStats) == 0 &&
		len(p.RecipeStats) == 0 &&
		len(p.Inventory) == 0 &&
		len(p.KnownRecipes) == 0 &&
		len(p.KnownStations) == 0 &&
		len(p.KnownMaterials) == 0 &&
		len(p.ShownTutorials) == 0 &&
		len(p.Uniques) == 0 &&
		len(p.Trophies) == 0 &&
		len(p.KnownBiomes) == 0 &&
		len(p.PlayerKnownTexts) == 0 &&
		len(p.Foods) == 0 &&
		len(p.Skills) == 0 &&
		len(p.CustomData) == 0
}
