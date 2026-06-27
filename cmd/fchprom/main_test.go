package main

import (
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/lanchelms/fch-decoder/valheim"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestParseCLIAcceptsLegacyComposeFlags(t *testing.T) {
	cli, err := parseCLI([]string{"-dir", "/characters", "-addr", ":9108"}, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cli.Dir != "/characters" || cli.Addr != ":9108" || cli.MetricsPath != "/metrics" || cli.Workers < 1 || cli.CacheTTL != defaultCacheTTL {
		t.Fatalf("cli = %+v", cli)
	}
}

func TestParseCLIAcceptsKongFlags(t *testing.T) {
	cli, err := parseCLI([]string{"--dir", "/characters", "--metrics-path", "/custom", "--workers", "2", "--cache-ttl", "10s"}, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cli.Dir != "/characters" || cli.MetricsPath != "/custom" || cli.Workers != 2 || cli.CacheTTL.String() != "10s" {
		t.Fatalf("cli = %+v", cli)
	}
}

func TestCharacterFilesFiltersBackups(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"Steam_111111_name.fch",
		"Steam_111111_name_backup_auto-638856016000.fch",
		"Steam_111111_name.fch.old",
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
	if len(got) != 1 || filepath.Base(got[0]) != "Steam_111111_name.fch" {
		t.Fatalf("characterFiles() = %#v, want only Steam_111111_name.fch", got)
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
	for _, name := range []string{"Deaths", "Builds", "EnemyKills", "BossKills"} {
		if !allowedPlayerStats[name] {
			t.Fatalf("allowedPlayerStats[%q] = false, want true", name)
		}
	}
	for _, name := range []string{"DistanceTraveled", "DistanceWalk", "DistanceRun", "DistanceSail", "DistanceAir", "SetGuardianPower", "SetPowerEikthyr", "UseGuardianPower", "UsePowerEikthyr", "Cheats", "DeathByEnemyHit", "DeathByFall"} {
		if allowedPlayerStats[name] {
			t.Fatalf("allowedPlayerStats[%q] = true, want false", name)
		}
	}
}

func TestCollectorMetricFamiliesHaveExpectedShape(t *testing.T) {
	character := valheim.NewCharacter("Fenris", 123)
	character.Player.Skills = []valheim.Skill{
		{Name: "Run", DisplayLevel: 34},
	}
	character.Player.RecipeStats = []valheim.StatEntry{
		{Name: "$item_arrow_fire", Value: 12},
	}
	character.Player.EnemyStats = []valheim.StatEntry{
		{Name: "$enemy_greyling", Value: 7},
	}
	character.PlayerStats = []valheim.StatEntry{
		{Name: "Deaths", Value: 3},
		{Name: "DistanceTraveled", Value: 456},
		{Name: "DistanceWalk", Value: 100},
		{Name: "DistanceRun", Value: 200},
		{Name: "DistanceAir", Value: 50},
		{Name: "DistanceSail", Value: 999},
	}

	c := &collector{
		cacheTTL: time.Hour,
		cachedAt: time.Now(),
		cached: snapshot{
			errors:     2,
			characters: []metrics{newMetrics(character)},
		},
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(c)

	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"valheim_character_skills":        {"player", "skill"},
		"valheim_character_crafting":      {"player", "recipe"},
		"valheim_character_enemies":       {"player", "enemy"},
		"valheim_character_stats":         {"player", "stat"},
		"valheim_character_distance":      {"player", "mode"},
		"valheim_character_scrape_errors": nil,
	}
	wantCount := map[string]int{
		"valheim_character_skills":        1,
		"valheim_character_crafting":      1,
		"valheim_character_enemies":       1,
		"valheim_character_stats":         1,
		"valheim_character_distance":      5,
		"valheim_character_scrape_errors": 1,
	}
	got := metricFamilies(families)
	if len(got) != len(want) {
		t.Fatalf("metric families = %v, want %v", sortedKeys(got), sortedKeys(want))
	}

	for name, labels := range want {
		family, ok := got[name]
		if !ok {
			t.Fatalf("metric family %q missing from %v", name, sortedKeys(got))
		}
		if family.GetType() != dto.MetricType_GAUGE {
			t.Fatalf("%s type = %s, want GAUGE", name, family.GetType())
		}
		if len(family.Metric) != wantCount[name] {
			t.Fatalf("%s metrics = %d, want %d", name, len(family.Metric), wantCount[name])
		}
		if gotLabels := labelNames(family.Metric[0]); !sameStringSet(gotLabels, labels) {
			t.Fatalf("%s labels = %v, want %v", name, gotLabels, labels)
		}
	}

	assertMetricValue(t, got["valheim_character_skills"], 34, map[string]string{"player": "Fenris", "skill": "Run"})
	assertMetricValue(t, got["valheim_character_crafting"], 12, map[string]string{"player": "Fenris", "recipe": "ArrowFire"})
	assertMetricValue(t, got["valheim_character_enemies"], 7, map[string]string{"player": "Fenris", "enemy": "Greyling"})
	assertMetricValue(t, got["valheim_character_stats"], 3, map[string]string{"player": "Fenris", "stat": "Deaths"})
	assertMetricValue(t, got["valheim_character_distance"], 456, map[string]string{"player": "Fenris", "mode": "Total"})
	assertMetricValue(t, got["valheim_character_distance"], 106, map[string]string{"player": "Fenris", "mode": "Sail"})
	assertMetricValue(t, got["valheim_character_scrape_errors"], 2, nil)
}

func TestDistanceMetricsInferSailingDistance(t *testing.T) {
	character, err := loadMetrics(filepath.Join("..", "..", "testdata", "Steam_333333_tugen.fch"))
	if err != nil {
		t.Fatal(err)
	}

	distances := map[string]float64{}
	for _, sample := range character.samples {
		if sample.desc == distanceDesc {
			distances[sample.labels[1]] = sample.value
		}
		if sample.desc == statsDesc && strings.HasPrefix(sample.labels[1], "Distance") {
			t.Fatalf("raw distance stat %q exported through stats metric", sample.labels[1])
		}
	}

	want := map[string]float64{
		"Total": 537925.5,
		"Walk":  210212.046875,
		"Run":   240471.453125,
		"Sail":  45461.73046875,
		"Air":   41780.26953125,
	}
	for mode, wantValue := range want {
		if math.Abs(distances[mode]-wantValue) > 0.001 {
			t.Fatalf("distance %s = %v, want %v", mode, distances[mode], wantValue)
		}
	}
}

func metricFamilies(families []*dto.MetricFamily) map[string]*dto.MetricFamily {
	got := make(map[string]*dto.MetricFamily, len(families))
	for _, family := range families {
		got[family.GetName()] = family
	}
	return got
}

func labelNames(metric *dto.Metric) []string {
	names := make([]string, 0, len(metric.Label))
	for _, label := range metric.Label {
		names = append(names, label.GetName())
	}
	return names
}

func assertMetricValue(t *testing.T, family *dto.MetricFamily, wantValue float64, labels map[string]string) {
	t.Helper()
	for _, metric := range family.Metric {
		if !metricLabelsMatch(metric, labels) {
			continue
		}
		if got := metric.GetGauge().GetValue(); got != wantValue {
			t.Fatalf("%s%v = %v, want %v", family.GetName(), labels, got, wantValue)
		}
		return
	}
	t.Fatalf("%s missing labels %v", family.GetName(), labels)
}

func metricLabelsMatch(metric *dto.Metric, labels map[string]string) bool {
	if len(metric.Label) != len(labels) {
		return false
	}
	for _, label := range metric.Label {
		if labels[label.GetName()] != label.GetValue() {
			return false
		}
	}
	return true
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sameStringSet(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Strings(a)
	sort.Strings(b)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
		for _, desc := range []*prometheus.Desc{skillsDesc, craftingDesc, enemiesDesc, statsDesc, distanceDesc} {
			if !seen[desc] {
				t.Fatalf("character %q is missing metrics for %v", character.player, desc)
			}
		}
	}
}
