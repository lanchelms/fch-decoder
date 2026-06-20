package fch

import (
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
			file:            "Steam_76561197968487130_fenris bueller.fch",
			name:            "Fenris Bueller",
			playerID:        3289368200,
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
			file:            "Steam_76561198018104185_bortson.fch",
			name:            "Bortson",
			playerID:        1835310974,
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
			if len(got.Player.KnownWorldKeys) == 0 {
				t.Fatal("KnownWorldKeys is empty")
			}
			if got.Player.GuardianPower.Name != "GP_Eikthyr" {
				t.Fatalf("GuardianPower = %q", got.Player.GuardianPower.Name)
			}
			if !got.Player.HasPlayerData {
				t.Fatal("HasPlayerData = false, want true")
			}
			if got.Player.PlayerDataLength != tt.playerLength {
				t.Fatalf("PlayerDataLength = %d, want %d", got.Player.PlayerDataLength, tt.playerLength)
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
			if got.RemainingBytes != 0 {
				t.Fatalf("RemainingBytes = %d, want 0", got.RemainingBytes)
			}
		})
	}
}

func TestSkillName(t *testing.T) {
	tests := map[int32]string{
		0:   "None",
		1:   "Swords",
		2:   "Knives",
		3:   "Clubs",
		4:   "Polearms",
		5:   "Spears",
		6:   "Blocking",
		7:   "Axes",
		8:   "Bows",
		9:   "ElementalMagic",
		10:  "BloodMagic",
		11:  "Unarmed",
		12:  "Pickaxes",
		13:  "WoodCutting",
		14:  "Crossbows",
		100: "Jump",
		101: "Sneak",
		102: "Run",
		103: "Swim",
		104: "Fishing",
		105: "Cooking",
		106: "Farming",
		107: "Crafting",
		108: "Dodge",
		110: "Ride",
		999: "All",
	}

	for skillType, want := range tests {
		if got := skillName(skillType); got != want {
			t.Fatalf("skillName(%d) = %q, want %q", skillType, got, want)
		}
	}

	if got := skillName(109); got != "" {
		t.Fatalf("skillName(109) = %q, want empty", got)
	}
}
