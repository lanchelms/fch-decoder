package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fch "github.com/lanchelms/fch-decoder"
)

func TestRunAppliesRepeatedEditFlags(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	out := filepath.Join(t.TempDir(), "edited.fch")

	var stdout, stderr bytes.Buffer
	err := run([]string{
		"-out", out,
		"-add-inventory", "Wood,stack=50,grid-x=1,grid-y=2,quality=3,crafter-name=Tester",
		"-add-inventory", "Stone,stack=25",
		"-remove-inventory", "Wood",
		"-set-skill-level", "Swords=44.5",
		"-set-skill-level", "Run=22",
		"-set-enemy-stat", "$enemy_greydwarf=123",
		"-set-enemy-stat", "$enemy_troll=9",
		"-set-material-stat", "$item_wood=777",
		"-set-material-stat", "$item_stone=12",
		"-set-player-stat", "Deaths=5",
		"-set-player-stat", "Builds=6",
		in,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run error = %v, stderr = %s", err, stderr.String())
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

	err := run([]string{"-in-place", "-set-player-stat", "Deaths=99", in}, ioDiscard{}, ioDiscard{})
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

func TestRunRequiresExplicitOutput(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")

	err := run([]string{"-set-player-stat", "Deaths=1", in}, ioDiscard{}, ioDiscard{})
	if err == nil || err.Error() != "missing required -out flag; use -in-place to overwrite the input file" {
		t.Fatalf("run error = %v, want missing output", err)
	}
}

func TestRunRejectsUnknownRemove(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	out := filepath.Join(t.TempDir(), "edited.fch")

	err := run([]string{"-out", out, "-remove-inventory", "DefinitelyMissing", in}, ioDiscard{}, ioDiscard{})
	if err == nil || err.Error() != `inventory item "DefinitelyMissing" not found` {
		t.Fatalf("run error = %v, want missing inventory item", err)
	}
}

func TestRunRejectsUnknownSkill(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	out := filepath.Join(t.TempDir(), "edited.fch")

	err := run([]string{"-out", out, "-set-skill-level", "Nope=1", in}, ioDiscard{}, ioDiscard{})
	if err == nil || !strings.Contains(err.Error(), `unknown skill "Nope"`) {
		t.Fatalf("run error = %v, want unknown skill", err)
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
