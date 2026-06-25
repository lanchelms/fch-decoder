package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fch "github.com/lanchelms/fch-decoder"
)

func TestRunAppliesEditCommands(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	out := filepath.Join(t.TempDir(), "edited.fch")

	var stdout, stderr bytes.Buffer
	commands := [][]string{
		{"--character", in, "--out", out, "add", "inventory", "Wood,stack=50,grid-x=1,grid-y=2,quality=3,crafter-name=Tester"},
		{"--character", out, "add", "inventory", "Stone,stack=25"},
		{"--character", out, "remove", "inventory", "Wood"},
		{"--character", out, "set", "skill", "Swords", "44.5"},
		{"--character", out, "set", "skill", "Run", "22"},
		{"--character", out, "set", "enemy", "$enemy_greydwarf", "123"},
		{"--character", out, "set", "enemy", "$enemy_troll", "9"},
		{"--character", out, "set", "material", "$item_wood", "777"},
		{"--character", out, "set", "material", "$item_stone", "12"},
		{"--character", out, "set", "player-stat", "Deaths", "5"},
		{"--character", out, "set", "player-stat", "Builds", "6"},
	}
	for _, args := range commands {
		if err := run(args, &stdout, &stderr); err != nil {
			t.Fatalf("run(%v) error = %v, stderr = %s", args, err, stderr.String())
		}
	}
	if !strings.Contains(stdout.String(), "wrote "+out) {
		t.Fatalf("stdout = %q, want wrote message", stdout.String())
	}

	got := decodeFile(t, out)
	if !got.Trailer.HashValid {
		t.Fatal("Trailer.HashValid = false, want true")
	}
	if findItem(got.Player.Inventory, "Wood") != nil {
		t.Fatal("Wood inventory item still present after add then remove")
	}
	stone := findItem(got.Player.Inventory, "Stone")
	if stone == nil {
		t.Fatal("Stone inventory item was not added")
	}
	if stone.Stack != 25 || stone.Durability != 1 || stone.Quality != 1 || !stone.PickedUp {
		t.Fatalf("Stone item = %+v, want stack 25 with defaults", *stone)
	}
	if got := skillLevel(got.Player.Skills, 1); got != 44.5 {
		t.Fatalf("Swords level = %v, want 44.5", got)
	}
	if got := skillLevel(got.Player.Skills, 102); got != 22 {
		t.Fatalf("Run level = %v, want 22", got)
	}
	if got := statValue(got.Player.EnemyStats, "$enemy_greydwarf"); got != 123 {
		t.Fatalf("enemy greydwarf stat = %v, want 123", got)
	}
	if got := statValue(got.Player.EnemyStats, "$enemy_troll"); got != 9 {
		t.Fatalf("enemy troll stat = %v, want 9", got)
	}
	if got := statValue(got.Player.MaterialStats, "$item_wood"); got != 777 {
		t.Fatalf("material wood stat = %v, want 777", got)
	}
	if got := statValue(got.Player.MaterialStats, "$item_stone"); got != 12 {
		t.Fatalf("material stone stat = %v, want 12", got)
	}
	if got := got.PlayerStats[0].Value; got != 5 {
		t.Fatalf("Deaths player stat = %v, want 5", got)
	}
	if got := got.PlayerStats[2].Value; got != 6 {
		t.Fatalf("Builds player stat = %v, want 6", got)
	}
}

func TestRunInPlace(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")

	err := run([]string{"--character", in, "set", "player-stat", "Deaths", "99"}, ioDiscard{}, ioDiscard{})
	if err != nil {
		t.Fatal(err)
	}

	got := decodeFile(t, in)
	if got.PlayerStats[0].Value != 99 {
		t.Fatalf("Deaths player stat = %v, want 99", got.PlayerStats[0].Value)
	}
	if !got.Trailer.HashValid {
		t.Fatal("Trailer.HashValid = false, want true")
	}
}

func TestRunUsesCharacterEnv(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	t.Setenv(characterEnv, in)

	if err := run([]string{"set", "player-stat", "Deaths", "77"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatal(err)
	}

	got := decodeFile(t, in)
	if got.PlayerStats[0].Value != 77 {
		t.Fatalf("Deaths player stat = %v, want 77", got.PlayerStats[0].Value)
	}
}

func TestRunPrefersExplicitCharacter(t *testing.T) {
	envPath := copyFixture(t, "Steam_333333_tugen.fch")
	explicitPath := copyFixture(t, "Steam_333333_tugen.fch")
	t.Setenv(characterEnv, envPath)

	if err := run([]string{"--character", explicitPath, "set", "player-stat", "Deaths", "88"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatal(err)
	}

	envCharacter := decodeFile(t, envPath)
	if envCharacter.PlayerStats[0].Value == 88 {
		t.Fatal("environment character was edited despite explicit character argument")
	}
	explicitCharacter := decodeFile(t, explicitPath)
	if explicitCharacter.PlayerStats[0].Value != 88 {
		t.Fatalf("explicit character Deaths = %v, want 88", explicitCharacter.PlayerStats[0].Value)
	}
}

func TestRunRequiresCharacter(t *testing.T) {
	err := run([]string{"set", "player-stat", "Deaths", "1"}, ioDiscard{}, ioDiscard{})
	if err == nil || !strings.Contains(err.Error(), "missing character") {
		t.Fatalf("run error = %v, want missing character", err)
	}
}

func TestRunChainsSingleEditCommands(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	out := filepath.Join(t.TempDir(), "edited.fch")

	commands := [][]string{
		{"--character", in, "--out", out, "add", "inventory", "Wood,stack=1"},
		{"--character", out, "remove", "inventory", "Wood"},
		{"--character", out, "add", "inventory", "Wood,stack=2"},
	}
	for _, args := range commands {
		if err := run(args, ioDiscard{}, ioDiscard{}); err != nil {
			t.Fatalf("run(%v) error = %v", args, err)
		}
	}

	got := decodeFile(t, out)
	wood := findItem(got.Player.Inventory, "Wood")
	if wood == nil || wood.Stack != 2 {
		t.Fatalf("Wood item = %+v, want final add to remain", wood)
	}
}

func TestRunWritesInPlaceByDefault(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")

	if err := run([]string{"--character", in, "set", "player-stat", "Deaths", "1"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatal(err)
	}
	got := decodeFile(t, in)
	if got.PlayerStats[0].Value != 1 {
		t.Fatalf("Deaths player stat = %v, want 1", got.PlayerStats[0].Value)
	}
}

func TestRunRejectsUnknownRemove(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	out := filepath.Join(t.TempDir(), "edited.fch")

	err := run([]string{"--character", in, "--out", out, "remove", "inventory", "DefinitelyMissing"}, ioDiscard{}, ioDiscard{})
	if err == nil || err.Error() != `inventory item "DefinitelyMissing" not found` {
		t.Fatalf("run error = %v, want missing inventory item", err)
	}
}

func TestRunRejectsUnknownSkill(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	out := filepath.Join(t.TempDir(), "edited.fch")

	err := run([]string{"--character", in, "--out", out, "set", "skill", "Nope", "1"}, ioDiscard{}, ioDiscard{})
	if err == nil || !strings.Contains(err.Error(), `unknown skill "Nope"`) {
		t.Fatalf("run error = %v, want unknown skill", err)
	}
}

func TestRunCreatesInPlaceBackup(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")

	if err := run([]string{"--character", in, "set", "player-stat", "Deaths", "3"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(in + ".bak"); err != nil {
		t.Fatalf("backup stat error = %v", err)
	}
}

func TestRunNoBackup(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")

	if err := run([]string{"--character", in, "--no-backup", "set", "player-stat", "Deaths", "3"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(in + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("backup stat error = %v, want not exist", err)
	}
}

func TestRunDryRunDoesNotWrite(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")

	var stdout bytes.Buffer
	if err := run([]string{"--character", in, "--dry-run", "set", "player-stat", "Deaths", "3"}, &stdout, ioDiscard{}); err != nil {
		t.Fatal(err)
	}
	got := decodeFile(t, in)
	if got.PlayerStats[0].Value == 3 {
		t.Fatal("dry run wrote the character file")
	}
	if !strings.Contains(stdout.String(), "would set player-stat Deaths=3") {
		t.Fatalf("stdout = %q, want dry-run summary", stdout.String())
	}
	if _, err := os.Stat(in + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("backup stat error = %v, want not exist", err)
	}
}

func TestRunListsWithoutCharacter(t *testing.T) {
	var stdout bytes.Buffer
	if err := run([]string{"list", "player-stats"}, &stdout, ioDiscard{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "0    Deaths") {
		t.Fatalf("stdout = %q, want player stats", stdout.String())
	}

	stdout.Reset()
	if err := run([]string{"list", "skills"}, &stdout, ioDiscard{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Run") {
		t.Fatalf("stdout = %q, want skills", stdout.String())
	}
}

func TestRunListsInventory(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")

	var stdout bytes.Buffer
	if err := run([]string{"--character", in, "list", "inventory"}, &stdout, ioDiscard{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "stack=") {
		t.Fatalf("stdout = %q, want inventory", stdout.String())
	}
}

func TestListInventoryAlignsColumns(t *testing.T) {
	runner := &editRunner{stdout: &bytes.Buffer{}}
	character := &fch.Character{
		Player: fch.PlayerData{
			Inventory: []fch.Item{
				{Name: "Wood", Stack: 1, Quality: 1, Durability: 1},
				{Name: "ReallyLongInventoryItemName", Stack: 2, Quality: 3, Durability: 4},
			},
		},
	}
	var stdout bytes.Buffer
	runner.stdout = &stdout

	if err := runner.listInventory(character); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("stdout = %q, want two inventory lines", stdout.String())
	}
	firstStack := strings.Index(lines[0], "stack=")
	secondStack := strings.Index(lines[1], "stack=")
	if firstStack == -1 || firstStack != secondStack {
		t.Fatalf("stdout = %q, want aligned stack columns", stdout.String())
	}
}

func TestRunRejectsUnsafeValues(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	tests := [][]string{
		{"--character", in, "set", "player-stat", "2147483647", "1"},
		{"--character", in, "set", "player-stat", "Deaths", "-1"},
		{"--character", in, "set", "skill", "Run", "101"},
		{"--character", in, "set", "skill", "Run", "NaN"},
		{"--character", in, "set", "enemy", " ", "1"},
		{"--character", in, "set", "material", "$item_wood", "-1"},
		{"--character", in, "add", "inventory", "Wood,stack=0"},
	}
	for _, args := range tests {
		if err := run(args, ioDiscard{}, ioDiscard{}); err == nil {
			t.Fatalf("run(%v) error = nil, want validation error", args)
		}
	}
}

func copyFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func decodeFile(t *testing.T, path string) *fch.Character {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := fch.DecodeBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func findItem(items []fch.Item, name string) *fch.Item {
	for i := range items {
		if items[i].Name == name {
			return &items[i]
		}
	}
	return nil
}

func skillLevel(skills []fch.Skill, skillType int32) float32 {
	for _, skill := range skills {
		if skill.Type == skillType {
			return skill.Level
		}
	}
	return 0
}

func statValue(entries []fch.StatEntry, name string) float32 {
	for _, entry := range entries {
		if entry.Name == name {
			return entry.Value
		}
	}
	return 0
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
