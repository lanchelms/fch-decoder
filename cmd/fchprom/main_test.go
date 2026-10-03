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

	fch "github.com/lanchelms/fch-decoder"
	"github.com/lanchelms/fch-decoder/valheim"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestParseCLIAcceptsKongFlags(t *testing.T) {
	cli, err := parseCLI([]string{"--dir", "/characters", "--metrics-path", "/custom", "--workers", "2", "--cache-ttl", "10s"}, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cli.Dir != "/characters" || cli.Addr != ":9108" || cli.MetricsPath != "/custom" || cli.Workers != 2 || cli.CacheTTL.String() != "10s" {
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
	character.Player.Health = 45
	character.Player.MaxHealth = 100
	character.Player.Stamina = 23
	character.Player.MaxStamina = 75
	character.Player.Eitr = 8
	character.Player.MaxEitr = 50
	character.Player.TimeSinceDeath = 89
	character.Player.GuardianPower.Cooldown = 321
	character.Player.KnownWorlds = []valheim.TimeEntry{
		{Name: "Meadows", Seconds: 1234},
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
		"valheim_character_stats":         {"player", "group", "stat"},
		"valheim_character_distance":      {"player", "mode"},
		"valheim_character":               {"player", "state"},
		"valheim_character_worlds":        {"player", "world"},
		"valheim_character_scrape_errors": nil,
	}
	wantCount := map[string]int{
		"valheim_character_skills":        1,
		"valheim_character_crafting":      1,
		"valheim_character_enemies":       1,
		"valheim_character_stats":         1,
		"valheim_character_distance":      5,
		"valheim_character":               8,
		"valheim_character_worlds":        1,
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
	assertMetricValue(t, got["valheim_character_stats"], 3, map[string]string{"player": "Fenris", "group": "RawStats", "stat": "Deaths"})
	assertMetricValue(t, got["valheim_character_distance"], 456, map[string]string{"player": "Fenris", "mode": "Total"})
	assertMetricValue(t, got["valheim_character_distance"], 106, map[string]string{"player": "Fenris", "mode": "Sail"})
	assertMetricValue(t, got["valheim_character"], 45, map[string]string{"player": "Fenris", "state": "Health"})
	assertMetricValue(t, got["valheim_character"], 89, map[string]string{"player": "Fenris", "state": "TimeSinceDeath"})
	assertMetricValue(t, got["valheim_character"], 321, map[string]string{"player": "Fenris", "state": "GuardianPowerCooldown"})
	assertMetricValue(t, got["valheim_character_worlds"], 1234, map[string]string{"player": "Fenris", "world": "Meadows"})
	assertMetricValue(t, got["valheim_character_scrape_errors"], 2, nil)
}

func TestDistanceMetricsInferSailingDistance(t *testing.T) {
	character, err := fch.DecodeFile(filepath.Join("..", "..", "testdata", "Steam_333333_tugen.fch"))
	if err != nil {
		t.Fatal(err)
	}

	distances := map[string]float64{}
	for _, sample := range newMetrics(character).samples {
		if sample.desc == distanceDesc {
			distances[sample.labels[1]] = sample.value
		}
		if sample.desc == statsDesc && strings.HasPrefix(sample.labels[2], "Distance") {
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
	if len(snap.characters) != 4 {
		t.Fatalf("snapshot characters = %d, want 4", len(snap.characters))
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
			if len(sample.labels) == 0 || sample.labels[0] != character.player {
				t.Fatalf("metric has labels %v, want player first", sample.labels)
			}
			for _, label := range sample.labels[1:] {
				if strings.Contains(label, "$") {
					t.Fatalf("metric label %q contains $", label)
				}
			}
			if sample.desc == skillsDesc && sample.value != math.Floor(sample.value) {
				t.Fatalf("skill metric %q = %v, want integer", sample.labels[1], sample.value)
			}
		}
		descs := []*prometheus.Desc{skillsDesc, craftingDesc, enemiesDesc, statsDesc, distanceDesc, characterDesc, knownWorldsDesc}
		if character.player == "Nichael" {
			descs = []*prometheus.Desc{skillsDesc, statsDesc, characterDesc, enemyKillsDesc, itemsCraftedDesc, worldTimeDesc}
		}
		for _, desc := range descs {
			if !seen[desc] {
				t.Fatalf("character %q is missing metrics for %v", character.player, desc)
			}
		}
	}
}

func TestMixedVersionRegisteredCollector(t *testing.T) {
	c := &collector{cacheTTL: time.Hour, cachedAt: time.Now(), cached: loadSnapshot(filepath.Join("..", "..", "testdata"), 2)}
	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(c)
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	got := metricFamilies(families)
	stats := got["valheim_character_stats"]
	groups := map[string]int{}
	for _, metric := range stats.Metric {
		labels := map[string]string{}
		for _, label := range metric.Label {
			labels[label.GetName()] = label.GetValue()
		}
		if labels["player"] == "Nichael" {
			groups[labels["group"]]++
		} else if labels["group"] != "RawStats" {
			t.Fatal("legacy group missing")
		}
	}
	if len(groups) != 3 || groups["RawStats"] != 205 || groups["Any"] != 205 || groups["Default"] != 205 {
		t.Fatalf("groups = %v", groups)
	}
	for g, n := range groups {
		if n != 205 {
			t.Fatalf("%s has %d scalars", g, n)
		}
	}
	for _, name := range []string{"crafting", "enemies", "distance", "worlds"} {
		for _, metric := range got["valheim_character_"+name].Metric {
			for _, label := range metric.Label {
				if label.GetName() == "player" && label.GetValue() == "Nichael" {
					t.Fatalf("duplicate compatibility family %s", name)
				}
			}
		}
	}
	assertMetricValue(t, stats, 34, map[string]string{"player": "Nichael", "group": "RawStats", "stat": "Deaths"})
	assertMetricValue(t, stats, 0, map[string]string{"player": "Nichael", "group": "Default", "stat": "Deaths"})
	for name, labels := range map[string][]string{
		"enemy_kills": {"player", "group", "modifier", "enemy"}, "items_crafted": {"player", "group", "item"}, "items_picked_up": {"player", "group", "item"}, "foods_eaten": {"player", "group", "food"}, "pieces_placed": {"player", "group", "piece"}, "world_time_seconds": {"player", "group", "world"}, "world_key_seconds": {"player", "group", "key", "setting"},
	} {
		family := got["valheim_character_"+name]
		if name == "pickables" {
			if family != nil {
				t.Fatal("fabricated pickables")
			}
			continue
		}
		if family == nil {
			t.Fatalf("missing %s", name)
		}
		if !sameStringSet(labelNames(family.Metric[0]), labels) {
			t.Fatalf("bad %s labels", name)
		}
	}
}

func TestGroupActivityAndWorldKeyMetrics(t *testing.T) {
	character, err := fch.DecodeFile(filepath.Join("..", "..", "testdata", "Steam_444444_nichael.fch"))
	if err != nil {
		t.Fatal(err)
	}
	character.StatGroups[6].Pickables = []valheim.StatEntry{{Name: "$raspberry", Value: 2}}
	for i := range character.StatGroups[6].EnemyStats {
		character.StatGroups[6].EnemyStats[i] = []valheim.StatEntry{{Name: "$enemy_greyling", Value: float32(i + 1)}}
	}
	c := &collector{cacheTTL: time.Hour, cachedAt: time.Now(), cached: snapshot{characters: []metrics{newMetrics(character)}}}
	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(c)
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	got := metricFamilies(families)
	assertMetricValue(t, got["valheim_character_pickables"], 2, map[string]string{"player": "Nichael", "group": "Default", "pickable": "raspberry"})
	for i, modifier := range []string{"MixedAndTotal", "Unarmed", "Magic", "Ranged", "Melee"} {
		assertMetricValue(t, got["valheim_character_enemy_kills"], float64(i+1), map[string]string{"player": "Nichael", "group": "Default", "modifier": modifier, "enemy": "Greyling"})
	}
	assertMetricValue(t, got["valheim_character_world_key_seconds"], 14673, map[string]string{"player": "Nichael", "group": "RawStats", "key": "nomap", "setting": ""})
}

func TestWorldKeyMetricsTrimAndSum(t *testing.T) {
	character := valheim.NewCharacter("Trim", 1)
	character.Version = 46
	character.StatGroups = []valheim.StatGroup{
		{Name: "RawStats", KnownWorldKeys: []valheim.WorldKey{
			valheim.NewWorldKey("nomap", 2),
			valheim.NewWorldKey("nomap ", 3),
			valheim.NewWorldKey("  nomap  ", 5),
			valheim.NewWorldKey(" nomap default ", 7),
			valheim.NewWorldKey("nomap default", 11),
			valheim.NewWorldKey("   ", 13),
		}},
		{Name: "Default", KnownWorldKeys: []valheim.WorldKey{valheim.NewWorldKey("nomap", 17)}},
	}
	c := &collector{cacheTTL: time.Hour, cachedAt: time.Now(), cached: snapshot{characters: []metrics{newMetrics(character)}}}
	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(c)
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	family := metricFamilies(families)["valheim_character_world_key_seconds"]
	if len(family.Metric) != 3 {
		t.Fatalf("expected three normalized series, got %d", len(family.Metric))
	}
	assertMetricValue(t, family, 10, map[string]string{"player": "Trim", "group": "RawStats", "key": "nomap", "setting": ""})
	assertMetricValue(t, family, 18, map[string]string{"player": "Trim", "group": "RawStats", "key": "nomap", "setting": "default"})
	assertMetricValue(t, family, 17, map[string]string{"player": "Trim", "group": "Default", "key": "nomap", "setting": ""})
	if character.StatGroups[0].KnownWorldKeys[2].Raw != "  nomap  " {
		t.Fatal("export changed the saved world key")
	}
}

func TestEmptyStatGroupMetrics(t *testing.T) {
	cases := []struct {
		name     string
		populate func(*valheim.StatGroup)
	}{
		{"empty", nil},
		{"scalar", func(g *valheim.StatGroup) { g.Stats[1].Value = 1 }},
		{"world", func(g *valheim.StatGroup) { g.KnownWorlds = []valheim.TimeEntry{{}} }},
		{"world key", func(g *valheim.StatGroup) { g.KnownWorldKeys = []valheim.WorldKey{{}} }},
		{"command", func(g *valheim.StatGroup) { g.KnownCommands = []valheim.StatEntry{{}} }},
		{"enemy", func(g *valheim.StatGroup) { g.EnemyStats[4] = []valheim.StatEntry{{}} }},
		{"picked up", func(g *valheim.StatGroup) { g.ItemsPickedUp = []valheim.StatEntry{{}} }},
		{"crafted", func(g *valheim.StatGroup) { g.ItemsCrafted = []valheim.StatEntry{{}} }},
		{"pickable", func(g *valheim.StatGroup) { g.Pickables = []valheim.StatEntry{{}} }},
		{"food", func(g *valheim.StatGroup) { g.FoodsEaten = []valheim.StatEntry{{}} }},
		{"piece", func(g *valheim.StatGroup) { g.PiecesPlaced = []valheim.StatEntry{{}} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			group := valheim.StatGroup{Name: "Hardcore", Stats: []valheim.StatEntry{{Name: "Deaths"}, {Name: "Jumps"}}, EnemyStats: make([][]valheim.StatEntry, 5)}
			if tc.populate != nil {
				tc.populate(&group)
			}
			m := metrics{player: "Test"}
			m.addGroups([]valheim.StatGroup{group})
			if tc.populate == nil {
				if len(m.samples) != 0 {
					t.Fatalf("empty group emitted %d samples", len(m.samples))
				}
				return
			}
			stats := 0
			for _, sample := range m.samples {
				if sample.desc == statsDesc {
					stats++
					if sample.labels[2] == "Deaths" && sample.value != 0 {
						t.Fatal("zero deaths changed")
					}
				}
			}
			if stats != 2 {
				t.Fatalf("populated group emitted %d scalars, want 2", stats)
			}
		})
	}
}
