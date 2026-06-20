package fch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeSamples(t *testing.T) {
	tests := []struct {
		file         string
		name         string
		playerID     uint64
		inventory    int
		materials    int
		recipes      int
		remainingMin int
	}{
		{
			file:         "Steam_76561197968487130_fenris bueller.fch",
			name:         "Fenris Bueller",
			playerID:     3289368200,
			inventory:    16,
			materials:    108,
			recipes:      35,
			remainingMin: 1000,
		},
		{
			file:         "Steam_76561198018104185_bortson.fch",
			name:         "Bortson",
			playerID:     1835310974,
			inventory:    16,
			materials:    141,
			recipes:      54,
			remainingMin: 1000,
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
			if got.PlayerDataVersion != 105 {
				t.Fatalf("PlayerDataVersion = %d, want 105", got.PlayerDataVersion)
			}
			if got.Player.Name != tt.name {
				t.Fatalf("Name = %q, want %q", got.Player.Name, tt.name)
			}
			if got.Player.PlayerID != tt.playerID {
				t.Fatalf("PlayerID = %d, want %d", got.Player.PlayerID, tt.playerID)
			}
			if len(got.Player.Worlds) != 1 {
				t.Fatalf("Worlds = %d, want 1", len(got.Player.Worlds))
			}
			if got.Player.Worlds[0].Name != "LanChelmsDeepNorth2" {
				t.Fatalf("World name = %q", got.Player.Worlds[0].Name)
			}
			if got.Player.GuardianPower.Name != "GP_Eikthyr" {
				t.Fatalf("GuardianPower = %q", got.Player.GuardianPower.Name)
			}
			if len(got.Player.Inventory) != tt.inventory {
				t.Fatalf("Inventory = %d, want %d", len(got.Player.Inventory), tt.inventory)
			}
			if got.Player.Inventory[0].Name == "" {
				t.Fatal("first inventory item has empty name")
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
			if got.Map.UncompressedLength == 0 {
				t.Fatal("map was not decompressed")
			}
			if got.Trailer.Length != 64 || len(got.Trailer.Hash) != 64 {
				t.Fatalf("bad trailer: length=%d hash=%d", got.Trailer.Length, len(got.Trailer.Hash))
			}
			if got.RemainingBytes < tt.remainingMin {
				t.Fatalf("RemainingBytes = %d, expected at least %d while tail sections are not decoded", got.RemainingBytes, tt.remainingMin)
			}
		})
	}
}
