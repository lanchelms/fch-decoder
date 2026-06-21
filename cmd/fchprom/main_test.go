package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestCharacterFilesFiltersBackups(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"Steam_123_name.fch",
		"Steam_123_name_backup_auto-638856016000.fch",
		"Steam_123_name.fch.old",
		"not-a-character.txt",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := characterFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || filepath.Base(got[0]) != "Steam_123_name.fch" {
		t.Fatalf("characterFiles() = %#v, want only Steam_123_name.fch", got)
	}
}

func TestCleanMetricLabel(t *testing.T) {
	tests := []struct {
		name string
		desc *prometheus.Desc
		want string
	}{
		{name: "$item_arrow_fire", desc: craftingDesc, want: "ArrowFire"},
		{name: "$enemy_greyling", desc: enemiesDesc, want: "Greyling"},
		{name: "$piece_trainingdummy", desc: enemiesDesc, want: "PieceTrainingdummy"},
		{name: "Deaths", desc: statsDesc, want: "Deaths"},
	}

	for _, tt := range tests {
		if got := cleanMetricLabel(tt.name, tt.desc); got != tt.want {
			t.Fatalf("cleanMetricLabel(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestAllowedPlayerStatsAreChoosy(t *testing.T) {
	for _, name := range []string{"Deaths", "Builds", "EnemyKills", "DistanceRun", "BossKills"} {
		if !allowedPlayerStats[name] {
			t.Fatalf("allowedPlayerStats[%q] = false, want true", name)
		}
	}
	for _, name := range []string{"SetGuardianPower", "SetPowerEikthyr", "UseGuardianPower", "UsePowerEikthyr", "Cheats", "DeathByEnemyHit", "DeathByFall"} {
		if allowedPlayerStats[name] {
			t.Fatalf("allowedPlayerStats[%q] = true, want false", name)
		}
	}
}

func TestLoadSnapshotFromFixtures(t *testing.T) {
	snap := loadSnapshot(filepath.Join("..", "..", "testdata"), 2)
	if snap.errors != 0 {
		t.Fatalf("snapshot errors = %d, want 0", snap.errors)
	}
	if len(snap.characters) != 3 {
		t.Fatalf("snapshot characters = %d, want 3", len(snap.characters))
	}

	for _, character := range snap.characters {
		if character.player == "" {
			t.Fatal("character has empty player name")
		}
		if len(character.samples) == 0 {
			t.Fatalf("character %q has no metrics", character.player)
		}
		seen := map[*prometheus.Desc]bool{}
		for _, sample := range character.samples {
			seen[sample.desc] = true
			if len(sample.labels) != 2 {
				t.Fatalf("metric has labels %v, want player and metric label", sample.labels)
			}
			if strings.Contains(sample.labels[1], "$") {
				t.Fatalf("metric label %q contains $", sample.labels[1])
			}
			if sample.desc == skillsDesc && sample.value != math.Floor(sample.value) {
				t.Fatalf("skill metric %q = %v, want integer", sample.labels[1], sample.value)
			}
		}
		for _, desc := range []*prometheus.Desc{skillsDesc, craftingDesc, enemiesDesc, statsDesc} {
			if !seen[desc] {
				t.Fatalf("character %q is missing metrics for %v", character.player, desc)
			}
		}
	}
}
