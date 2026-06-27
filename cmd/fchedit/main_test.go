package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lanchelms/fch-decoder"
	"github.com/lanchelms/fch-decoder/valheim"
)

func TestRunAppliesEditCommands(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	out := filepath.Join(t.TempDir(), "edited.fch")

	var stdout, stderr bytes.Buffer
	commands := [][]string{
		{"--character", in, "--out", out, "add", "inventory", "Wood,stack=50,pos=1:2,quality=1,crafter-name=Tester"},
		{"--character", out, "add", "inventory", "Stone,stack=25,pos=2:2"},
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

	got, err := fch.DecodeFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Trailer.HashValid {
		t.Fatal("Trailer.HashValid = false, want true")
	}
	if _, ok := got.InventoryItem("Wood"); ok {
		t.Fatal("Wood inventory item still present after add then remove")
	}
	stone, ok := got.InventoryItem("Stone")
	if !ok {
		t.Fatal("Stone inventory item was not added")
	}
	if stone.Stack != 25 || stone.Durability != 100 || stone.Quality != 1 || stone.GridX != 2 || stone.GridY != 2 || !stone.PickedUp {
		t.Fatalf("Stone item = %+v, want stack 25 with defaults", *stone)
	}
	if skill, ok := got.Skill(1); !ok || skill.Level != 44.5 {
		t.Fatalf("Swords skill = %+v ok=%v, want level 44.5", skill, ok)
	}
	if skill, ok := got.Skill(102); !ok || skill.Level != 22 {
		t.Fatalf("Run skill = %+v ok=%v, want level 22", skill, ok)
	}
	if value, ok := got.EnemyStat("$enemy_greydwarf"); !ok || value != 123 {
		t.Fatalf("enemy greydwarf stat = %v ok=%v, want 123", value, ok)
	}
	if value, ok := got.EnemyStat("$enemy_troll"); !ok || value != 9 {
		t.Fatalf("enemy troll stat = %v ok=%v, want 9", value, ok)
	}
	if value, ok := got.MaterialStat("$item_wood"); !ok || value != 777 {
		t.Fatalf("material wood stat = %v ok=%v, want 777", value, ok)
	}
	if value, ok := got.MaterialStat("$item_stone"); !ok || value != 12 {
		t.Fatalf("material stone stat = %v ok=%v, want 12", value, ok)
	}
	if got := got.PlayerStats[0].Value; got != 5 {
		t.Fatalf("Deaths player stat = %v, want 5", got)
	}
	if got := got.PlayerStats[2].Value; got != 6 {
		t.Fatalf("Builds player stat = %v, want 6", got)
	}
	if count := countCustomData(got.Player.CustomData, fcheditLastModifiedKey); count != 1 {
		t.Fatalf("%s custom data count = %d, want 1", fcheditLastModifiedKey, count)
	}
	requireFcheditLastModified(t, got)
}

func TestRunCreditsRecipeInventoryItem(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	out := filepath.Join(t.TempDir(), "edited.fch")

	var stdout, stderr bytes.Buffer
	args := []string{"--character", in, "--out", out, "add", "inventory", "SwordIron,pos=3:2"}
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatalf("run(%v) error = %v, stderr = %s", args, err, stderr.String())
	}

	got, err := fch.DecodeFile(out)
	if err != nil {
		t.Fatal(err)
	}
	item, ok := got.InventoryItem("SwordIron")
	if !ok {
		t.Fatal("SwordIron inventory item was not added")
	}
	if item.CrafterID != got.Player.PlayerID || item.CrafterName != got.Player.Name || !item.PickedUp {
		t.Fatalf("SwordIron item = %+v, want player crafter %d/%q and picked up", *item, got.Player.PlayerID, got.Player.Name)
	}
}

func TestRunShortFlags(t *testing.T) {
	t.Run("character and out", func(t *testing.T) {
		in := copyFixture(t, "Steam_333333_tugen.fch")
		out := filepath.Join(t.TempDir(), "edited.fch")

		if err := run([]string{"-c", in, "-o", out, "set", "player-stat", "Deaths", "4"}, io.Discard, io.Discard); err != nil {
			t.Fatal(err)
		}

		got, err := fch.DecodeFile(out)
		if err != nil {
			t.Fatal(err)
		}
		if got.PlayerStats[0].Value != 4 {
			t.Fatalf("Deaths player stat = %v, want 4", got.PlayerStats[0].Value)
		}
		requireFcheditLastModified(t, got)
	})

	t.Run("no backup", func(t *testing.T) {
		in := copyFixture(t, "Steam_333333_tugen.fch")

		if err := run([]string{"-c", in, "-n", "set", "player-stat", "Deaths", "5"}, io.Discard, io.Discard); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(in + ".bak"); !os.IsNotExist(err) {
			t.Fatalf("backup stat error = %v, want not exist", err)
		}
	})

	t.Run("dry run", func(t *testing.T) {
		in := copyFixture(t, "Steam_333333_tugen.fch")

		var stdout bytes.Buffer
		if err := run([]string{"-c", in, "-d", "set", "player-stat", "Deaths", "6"}, &stdout, io.Discard); err != nil {
			t.Fatal(err)
		}
		got, err := fch.DecodeFile(in)
		if err != nil {
			t.Fatal(err)
		}
		if got.PlayerStats[0].Value == 6 {
			t.Fatal("dry run wrote the character file")
		}
		if !strings.Contains(stdout.String(), "would set player-stat Deaths=6") {
			t.Fatalf("stdout = %q, want dry-run summary", stdout.String())
		}
	})
}

func TestRunEditsOnlyRequestedCategory(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		expect func(*valheim.Character)
	}{
		{
			name: "add inventory",
			args: []string{"add", "inventory", "Needle,stack=3,pos=4:1"},
			expect: func(character *valheim.Character) {
				character.Player.Inventory = append(character.Player.Inventory, valheim.Item{
					Name:       "Needle",
					Stack:      3,
					Durability: 100,
					GridX:      4,
					GridY:      1,
					Quality:    1,
					CustomData: []valheim.TextEntry{},
					PickedUp:   true,
				})
			},
		},
		{
			name: "remove inventory",
			args: []string{"remove", "inventory", "Hammer"},
			expect: func(character *valheim.Character) {
				if err := character.RemoveInventoryItem("Hammer"); err != nil {
					panic(err)
				}
			},
		},
		{
			name: "set skill",
			args: []string{"set", "skill", "Run", "22"},
			expect: func(character *valheim.Character) {
				character.SetSkill(102, 22)
			},
		},
		{
			name: "set enemy",
			args: []string{"set", "enemy", "$enemy_greydwarf", "123"},
			expect: func(character *valheim.Character) {
				character.UpsertEnemyStat("$enemy_greydwarf", 123)
			},
		},
		{
			name: "set material",
			args: []string{"set", "material", "$item_wood", "777"},
			expect: func(character *valheim.Character) {
				character.UpsertMaterialStat("$item_wood", 777)
			},
		},
		{
			name: "set player stat",
			args: []string{"set", "player-stat", "Deaths", "5"},
			expect: func(character *valheim.Character) {
				if err := character.SetPlayerStat(0, "Deaths", 5); err != nil {
					panic(err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := copyFixture(t, "Steam_333333_tugen.fch")
			before, err := fch.DecodeFile(in)
			if err != nil {
				t.Fatal(err)
			}
			oldHash := append([]byte(nil), before.Trailer.Hash...)

			args := append([]string{"--character", in, "--no-backup"}, tt.args...)
			if err := run(args, io.Discard, io.Discard); err != nil {
				t.Fatalf("run(%v) error = %v", args, err)
			}
			after, err := fch.DecodeFile(in)
			if err != nil {
				t.Fatal(err)
			}

			if !after.Trailer.HashValid {
				t.Fatal("Trailer.HashValid = false, want true")
			}
			if bytes.Equal(after.Trailer.Hash, oldHash) {
				t.Fatal("Trailer.Hash was not recalculated")
			}

			expected := before
			tt.expect(expected)
			expected.FileLength = after.FileLength
			expected.PlayerDataLength = after.PlayerDataLength
			expected.Trailer = after.Trailer
			expected.UpsertCustomData(fcheditLastModifiedKey, requireFcheditLastModified(t, after))

			if !reflect.DeepEqual(after, expected) {
				t.Fatal("decoded character changed outside the requested edit and encoder metadata")
			}
		})
	}
}

func TestRunInPlace(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")

	err := run([]string{"--character", in, "set", "player-stat", "Deaths", "99"}, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	got, err := fch.DecodeFile(in)
	if err != nil {
		t.Fatal(err)
	}
	if got.PlayerStats[0].Value != 99 {
		t.Fatalf("Deaths player stat = %v, want 99", got.PlayerStats[0].Value)
	}
	if !got.Trailer.HashValid {
		t.Fatal("Trailer.HashValid = false, want true")
	}
	requireFcheditLastModified(t, got)
}

func TestRunUsesCharacterEnv(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	t.Setenv(characterEnv, in)

	if err := run([]string{"set", "player-stat", "Deaths", "77"}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}

	got, err := fch.DecodeFile(in)
	if err != nil {
		t.Fatal(err)
	}
	if got.PlayerStats[0].Value != 77 {
		t.Fatalf("Deaths player stat = %v, want 77", got.PlayerStats[0].Value)
	}
	requireFcheditLastModified(t, got)
}

func TestRunPrefersExplicitCharacter(t *testing.T) {
	envPath := copyFixture(t, "Steam_333333_tugen.fch")
	explicitPath := copyFixture(t, "Steam_333333_tugen.fch")
	t.Setenv(characterEnv, envPath)

	if err := run([]string{"--character", explicitPath, "set", "player-stat", "Deaths", "88"}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}

	envCharacter, err := fch.DecodeFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if envCharacter.PlayerStats[0].Value == 88 {
		t.Fatal("environment character was edited despite explicit character argument")
	}
	explicitCharacter, err := fch.DecodeFile(explicitPath)
	if err != nil {
		t.Fatal(err)
	}
	if explicitCharacter.PlayerStats[0].Value != 88 {
		t.Fatalf("explicit character Deaths = %v, want 88", explicitCharacter.PlayerStats[0].Value)
	}
	requireFcheditLastModified(t, explicitCharacter)
}

func TestRunRequiresCharacter(t *testing.T) {
	oldCharacter, hadCharacter := os.LookupEnv(characterEnv)
	if err := os.Unsetenv(characterEnv); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if hadCharacter {
			if err := os.Setenv(characterEnv, oldCharacter); err != nil {
				t.Fatal(err)
			}
			return
		}
		if err := os.Unsetenv(characterEnv); err != nil {
			t.Fatal(err)
		}
	}()

	err := run([]string{"set", "player-stat", "Deaths", "1"}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "missing character") {
		t.Fatalf("run error = %v, want missing character", err)
	}
}

func TestRunChainsSingleEditCommands(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	out := filepath.Join(t.TempDir(), "edited.fch")

	commands := [][]string{
		{"--character", in, "--out", out, "add", "inventory", "Wood,stack=1,pos=2:2"},
		{"--character", out, "remove", "inventory", "Wood"},
		{"--character", out, "add", "inventory", "Wood,stack=2,pos=2:2"},
	}
	for _, args := range commands {
		if err := run(args, io.Discard, io.Discard); err != nil {
			t.Fatalf("run(%v) error = %v", args, err)
		}
	}

	got, err := fch.DecodeFile(out)
	if err != nil {
		t.Fatal(err)
	}
	wood, ok := got.InventoryItem("Wood")
	if !ok || wood.Stack != 2 {
		t.Fatalf("Wood item = %+v, want final add to remain", wood)
	}
}

func TestRunWritesInPlaceByDefault(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")

	if err := run([]string{"--character", in, "set", "player-stat", "Deaths", "1"}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	got, err := fch.DecodeFile(in)
	if err != nil {
		t.Fatal(err)
	}
	if got.PlayerStats[0].Value != 1 {
		t.Fatalf("Deaths player stat = %v, want 1", got.PlayerStats[0].Value)
	}
	requireFcheditLastModified(t, got)
}

func TestRunRejectsUnknownRemove(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	out := filepath.Join(t.TempDir(), "edited.fch")

	err := run([]string{"--character", in, "--out", out, "remove", "inventory", "DefinitelyMissing"}, io.Discard, io.Discard)
	if err == nil || err.Error() != `inventory item "DefinitelyMissing" not found` {
		t.Fatalf("run error = %v, want missing inventory item", err)
	}
}

func TestRunAddsInventoryAtNextEmptySlot(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	before, err := fch.DecodeFile(in)
	if err != nil {
		t.Fatal(err)
	}
	x, y, ok := before.EmptyInventorySlot()
	if !ok {
		t.Fatal("fixture has no empty inventory slot")
	}

	if err := run([]string{"--character", in, "add", "inventory", "Needle,stack=3"}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}

	after, err := fch.DecodeFile(in)
	if err != nil {
		t.Fatal(err)
	}
	item, ok := after.InventorySlot(x, y)
	if !ok || item.Name != "Needle" || item.Stack != 3 {
		t.Fatalf("placed item = %+v, want Needle stack 3 at %d:%d", item, x, y)
	}
}

func TestRunReplacesOccupiedInventorySlot(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	before, err := fch.DecodeFile(in)
	if err != nil {
		t.Fatal(err)
	}
	occupied := before.Player.Inventory[0]
	spec := fmt.Sprintf("Stone,stack=25,pos=%d:%d", occupied.GridX, occupied.GridY)

	if err := run([]string{"--character", in, "add", "inventory", spec}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}

	after, err := fch.DecodeFile(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Player.Inventory) != len(before.Player.Inventory) {
		t.Fatalf("Inventory = %d, want %d", len(after.Player.Inventory), len(before.Player.Inventory))
	}
	item, ok := after.InventorySlot(occupied.GridX, occupied.GridY)
	if !ok || item.Name != "Stone" || item.Stack != 25 || item.Durability != 100 {
		t.Fatalf("replaced item = %+v, want Stone stack 25 at occupied slot", item)
	}
}

func TestRunRejectsUnknownSkill(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	out := filepath.Join(t.TempDir(), "edited.fch")

	err := run([]string{"--character", in, "--out", out, "set", "skill", "Nope", "1"}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), `unknown skill "Nope"`) {
		t.Fatalf("run error = %v, want unknown skill", err)
	}
}

func TestRunCreatesInPlaceBackup(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	before, err := os.ReadFile(in)
	if err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := run([]string{"--character", in, "set", "player-stat", "Deaths", "3"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	backup := in + ".bak"
	if _, err := os.Stat(backup); err != nil {
		t.Fatalf("backup stat error = %v", err)
	}
	got, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, before) {
		t.Fatal("backup did not preserve the pre-edit character file")
	}
	if !strings.Contains(stdout.String(), "backup "+backup) {
		t.Fatalf("stdout = %q, want backup line", stdout.String())
	}
}

func TestRunReusesRecentInPlaceBackup(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	backup := in + ".bak"
	numberedBackup := in + ".bak.1"
	backupData := []byte("first rollback point")
	if err := os.WriteFile(backup, backupData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(backup, now.Add(-30*time.Minute), now.Add(-30*time.Minute)); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	err := runTimed([]string{"--character", in, "set", "player-stat", "Deaths", "3"}, &stdout, io.Discard, func() time.Time {
		return now
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, backupData) {
		t.Fatal("recent backup was overwritten")
	}
	if _, err := os.Stat(numberedBackup); !os.IsNotExist(err) {
		t.Fatalf("numbered backup stat error = %v, want not exist", err)
	}
	if strings.Contains(stdout.String(), "backup ") {
		t.Fatalf("stdout = %q, want no backup line", stdout.String())
	}
}

func TestRunOverwritesStaleInPlaceBackup(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	before, err := os.ReadFile(in)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	backup := in + ".bak"
	if err := os.WriteFile(backup, []byte("old rollback point"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(backup, now.Add(-time.Hour-time.Second), now.Add(-time.Hour-time.Second)); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	err = runTimed([]string{"--character", in, "set", "player-stat", "Deaths", "3"}, &stdout, io.Discard, func() time.Time {
		return now
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, before) {
		t.Fatal("stale backup was not overwritten with the pre-edit character file")
	}
	if !strings.Contains(stdout.String(), "backup "+backup) {
		t.Fatalf("stdout = %q, want backup line", stdout.String())
	}
}

func TestRunNoBackup(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")

	if err := run([]string{"--character", in, "--no-backup", "set", "player-stat", "Deaths", "3"}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(in + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("backup stat error = %v, want not exist", err)
	}
}

func TestRunDryRunDoesNotWrite(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")

	var stdout bytes.Buffer
	if err := run([]string{"--character", in, "--dry-run", "set", "player-stat", "Deaths", "3"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	got, err := fch.DecodeFile(in)
	if err != nil {
		t.Fatal(err)
	}
	if got.PlayerStats[0].Value == 3 {
		t.Fatal("dry run wrote the character file")
	}
	if value, ok := got.CustomData(fcheditLastModifiedKey); ok {
		t.Fatalf("dry run wrote %s custom data value %q", fcheditLastModifiedKey, value)
	}
	if !strings.Contains(stdout.String(), "would set player-stat Deaths=3") {
		t.Fatalf("stdout = %q, want dry-run summary", stdout.String())
	}
	if !strings.Contains(stdout.String(), "would set custom-data "+fcheditLastModifiedKey+"=") {
		t.Fatalf("stdout = %q, want dry-run custom data summary", stdout.String())
	}
	if _, err := os.Stat(in + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("backup stat error = %v, want not exist", err)
	}
}

func TestRunListsWithoutCharacter(t *testing.T) {
	var stdout bytes.Buffer
	if err := run([]string{"list", "player-stats"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "0    Deaths") {
		t.Fatalf("stdout = %q, want player stats", stdout.String())
	}

	stdout.Reset()
	if err := run([]string{"list", "skills"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Run") {
		t.Fatalf("stdout = %q, want skills", stdout.String())
	}

	stdout.Reset()
	if err := run([]string{"list", "items"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "SwordIron") || !strings.Contains(stdout.String(), "max-quality=4") {
		t.Fatalf("stdout = %q, want item metadata", stdout.String())
	}
	if strings.Contains(stdout.String(), "inventory-valid") {
		t.Fatalf("stdout = %q, want inventory-valid hidden", stdout.String())
	}
	if strings.Contains(stdout.String(), "Abomination_attack1") {
		t.Fatalf("stdout = %q, want internal items hidden", stdout.String())
	}

	stdout.Reset()
	if err := run([]string{"list", "items", "--all"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Abomination_attack1") || strings.Contains(stdout.String(), "inventory-valid") {
		t.Fatalf("stdout = %q, want all item metadata", stdout.String())
	}
}

func TestRunListsInventory(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")

	var stdout bytes.Buffer
	if err := run([]string{"--character", in, "list", "inventory"}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "stack=") {
		t.Fatalf("stdout = %q, want inventory", stdout.String())
	}
}

func TestListInventoryAlignsColumns(t *testing.T) {
	runner := &editRunner{stdout: &bytes.Buffer{}}
	character := &valheim.Character{
		Player: valheim.Player{
			PlayerState: valheim.PlayerState{
				Inventory: []valheim.Item{
					{Name: "Wood", Stack: 1, Quality: 1, Durability: 1},
					{Name: "ReallyLongInventoryItemName", Stack: 2, Quality: 3, Durability: 4},
				},
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
		if err := run(args, io.Discard, io.Discard); err == nil {
			t.Fatalf("run(%v) error = nil, want validation error", args)
		}
	}
}

func TestRunRejectsInvalidTrailerHash(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	before, err := os.ReadFile(in)
	if err != nil {
		t.Fatal(err)
	}
	data := append([]byte(nil), before...)
	data[12] ^= 1
	if err := os.WriteFile(in, data, 0o644); err != nil {
		t.Fatal(err)
	}

	err = run([]string{"--character", in, "--no-backup", "set", "player-stat", "Deaths", "1"}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "invalid trailer hash") {
		t.Fatalf("run error = %v, want invalid trailer hash", err)
	}
	got, err := os.ReadFile(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("fchedit modified a file with an invalid trailer hash")
	}
	if _, err := os.Stat(in + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("backup stat error = %v, want not exist", err)
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

func requireFcheditLastModified(t *testing.T, character *valheim.Character) string {
	t.Helper()
	value, ok := character.CustomData(fcheditLastModifiedKey)
	if !ok {
		t.Fatalf("missing %s custom data", fcheditLastModifiedKey)
	}
	if _, err := time.Parse(fcheditLastModifiedValue, value); err != nil {
		t.Fatalf("%s = %q, want %s timestamp: %v", fcheditLastModifiedKey, value, fcheditLastModifiedValue, err)
	}
	return value
}

func countCustomData(entries []valheim.TextEntry, key string) int {
	count := 0
	for _, entry := range entries {
		if entry.Key == key {
			count++
		}
	}
	return count
}
