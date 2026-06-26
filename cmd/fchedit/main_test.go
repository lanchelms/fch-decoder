package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	fch "github.com/lanchelms/fch-decoder"
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
	if stone.Stack != 25 || stone.Durability != 100 || stone.Quality != 1 || stone.GridX != 2 || stone.GridY != 2 || !stone.PickedUp {
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

	got := decodeFile(t, out)
	item := findItem(got.Player.Inventory, "SwordIron")
	if item == nil {
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

		if err := run([]string{"-c", in, "-o", out, "set", "player-stat", "Deaths", "4"}, ioDiscard{}, ioDiscard{}); err != nil {
			t.Fatal(err)
		}

		got := decodeFile(t, out)
		if got.PlayerStats[0].Value != 4 {
			t.Fatalf("Deaths player stat = %v, want 4", got.PlayerStats[0].Value)
		}
		requireFcheditLastModified(t, got)
	})

	t.Run("no backup", func(t *testing.T) {
		in := copyFixture(t, "Steam_333333_tugen.fch")

		if err := run([]string{"-c", in, "-n", "set", "player-stat", "Deaths", "5"}, ioDiscard{}, ioDiscard{}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(in + ".bak"); !os.IsNotExist(err) {
			t.Fatalf("backup stat error = %v, want not exist", err)
		}
	})

	t.Run("dry run", func(t *testing.T) {
		in := copyFixture(t, "Steam_333333_tugen.fch")

		var stdout bytes.Buffer
		if err := run([]string{"-c", in, "-d", "set", "player-stat", "Deaths", "6"}, &stdout, ioDiscard{}); err != nil {
			t.Fatal(err)
		}
		got := decodeFile(t, in)
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
		expect func(*fch.Character)
	}{
		{
			name: "add inventory",
			args: []string{"add", "inventory", "Needle,stack=3,pos=4:1"},
			expect: func(character *fch.Character) {
				character.Player.Inventory = append(character.Player.Inventory, fch.Item{
					Name:       "Needle",
					Stack:      3,
					Durability: 100,
					GridX:      4,
					GridY:      1,
					Quality:    1,
					CustomData: []fch.TextEntry{},
					PickedUp:   true,
				})
			},
		},
		{
			name: "remove inventory",
			args: []string{"remove", "inventory", "Hammer"},
			expect: func(character *fch.Character) {
				if err := character.RemoveInventoryItem("Hammer"); err != nil {
					panic(err)
				}
			},
		},
		{
			name: "set skill",
			args: []string{"set", "skill", "Run", "22"},
			expect: func(character *fch.Character) {
				character.SetSkill(102, 22)
			},
		},
		{
			name: "set enemy",
			args: []string{"set", "enemy", "$enemy_greydwarf", "123"},
			expect: func(character *fch.Character) {
				character.UpsertEnemyStat("$enemy_greydwarf", 123)
			},
		},
		{
			name: "set material",
			args: []string{"set", "material", "$item_wood", "777"},
			expect: func(character *fch.Character) {
				character.UpsertMaterialStat("$item_wood", 777)
			},
		},
		{
			name: "set player stat",
			args: []string{"set", "player-stat", "Deaths", "5"},
			expect: func(character *fch.Character) {
				if err := character.SetPlayerStat(0, "Deaths", 5); err != nil {
					panic(err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := copyFixture(t, "Steam_333333_tugen.fch")
			before := decodeFile(t, in)
			oldHash := append([]byte(nil), before.Trailer.Hash...)

			args := append([]string{"--character", in, "--no-backup"}, tt.args...)
			if err := run(args, ioDiscard{}, ioDiscard{}); err != nil {
				t.Fatalf("run(%v) error = %v", args, err)
			}
			after := decodeFile(t, in)

			if !after.Trailer.HashValid {
				t.Fatal("Trailer.HashValid = false, want true")
			}
			if bytes.Equal(after.Trailer.Hash, oldHash) {
				t.Fatal("Trailer.Hash was not recalculated")
			}

			expected := before
			tt.expect(expected)
			expected.FileLength = after.FileLength
			expected.Player.PlayerDataLength = after.Player.PlayerDataLength
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
	requireFcheditLastModified(t, got)
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
	requireFcheditLastModified(t, got)
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

	err := run([]string{"set", "player-stat", "Deaths", "1"}, ioDiscard{}, ioDiscard{})
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
	requireFcheditLastModified(t, got)
}

func TestRunRejectsUnknownRemove(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	out := filepath.Join(t.TempDir(), "edited.fch")

	err := run([]string{"--character", in, "--out", out, "remove", "inventory", "DefinitelyMissing"}, ioDiscard{}, ioDiscard{})
	if err == nil || err.Error() != `inventory item "DefinitelyMissing" not found` {
		t.Fatalf("run error = %v, want missing inventory item", err)
	}
}

func TestRunAddsInventoryAtNextEmptySlot(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	before := decodeFile(t, in)
	x, y, ok := nextEmptyInventorySlot(before.Player.Inventory)
	if !ok {
		t.Fatal("fixture has no empty inventory slot")
	}

	if err := run([]string{"--character", in, "add", "inventory", "Needle,stack=3"}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatal(err)
	}

	after := decodeFile(t, in)
	item := findSlot(after.Player.Inventory, x, y)
	if item == nil || item.Name != "Needle" || item.Stack != 3 {
		t.Fatalf("placed item = %+v, want Needle stack 3 at %d:%d", item, x, y)
	}
}

func TestRunReplacesOccupiedInventorySlot(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	before := decodeFile(t, in)
	occupied := before.Player.Inventory[0]
	spec := fmt.Sprintf("Stone,stack=25,pos=%d:%d", occupied.GridX, occupied.GridY)

	if err := run([]string{"--character", in, "add", "inventory", spec}, ioDiscard{}, ioDiscard{}); err != nil {
		t.Fatal(err)
	}

	after := decodeFile(t, in)
	if len(after.Player.Inventory) != len(before.Player.Inventory) {
		t.Fatalf("Inventory = %d, want %d", len(after.Player.Inventory), len(before.Player.Inventory))
	}
	item := findSlot(after.Player.Inventory, occupied.GridX, occupied.GridY)
	if item == nil || item.Name != "Stone" || item.Stack != 25 || item.Durability != 100 {
		t.Fatalf("replaced item = %+v, want Stone stack 25 at occupied slot", item)
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
	if value, ok := customDataValue(got.Player.CustomData, fcheditLastModifiedKey); ok {
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

	stdout.Reset()
	if err := run([]string{"list", "items"}, &stdout, ioDiscard{}); err != nil {
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
	if err := run([]string{"list", "items", "--all"}, &stdout, ioDiscard{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Abomination_attack1") || strings.Contains(stdout.String(), "inventory-valid") {
		t.Fatalf("stdout = %q, want all item metadata", stdout.String())
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

func TestRunRejectsInvalidTrailerHash(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	before := readFile(t, in)
	data := append([]byte(nil), before...)
	data[12] ^= 1
	writeTestFile(t, in, data)

	err := run([]string{"--character", in, "--no-backup", "set", "player-stat", "Deaths", "1"}, ioDiscard{}, ioDiscard{})
	if err == nil || !strings.Contains(err.Error(), "invalid trailer hash") {
		t.Fatalf("run error = %v, want invalid trailer hash", err)
	}
	if got := readFile(t, in); !bytes.Equal(got, data) {
		t.Fatal("fchedit modified a file with an invalid trailer hash")
	}
	if _, err := os.Stat(in + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("backup stat error = %v, want not exist", err)
	}
}

func TestRunRejectsUnsupportedVersions(t *testing.T) {
	tests := []struct {
		name string
		edit func(*fch.Character)
		want string
	}{
		{
			name: "character version",
			edit: func(character *fch.Character) {
				character.Version++
			},
			want: "unsupported character version 44",
		},
		{
			name: "player version",
			edit: func(character *fch.Character) {
				character.Player.PlayerVersion++
			},
			want: "unsupported player version 30",
		},
		{
			name: "inventory version",
			edit: func(character *fch.Character) {
				character.Player.InventoryVersion++
			},
			want: "unsupported inventory version 107",
		},
		{
			name: "skill version",
			edit: func(character *fch.Character) {
				character.Player.SkillVersion++
			},
			want: "unsupported skill version 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := copyFixture(t, "Steam_333333_tugen.fch")
			character := decodeFile(t, in)
			tt.edit(character)
			data, err := fch.EncodeBytes(character)
			if err != nil {
				t.Fatal(err)
			}
			writeTestFile(t, in, data)

			err = run([]string{"--character", in, "--no-backup", "set", "player-stat", "Deaths", "1"}, ioDiscard{}, ioDiscard{})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("run error = %v, want %q", err, tt.want)
			}
			if got := readFile(t, in); !bytes.Equal(got, data) {
				t.Fatal("fchedit modified a file with an unsupported version")
			}
		})
	}
}

func TestRunRejectsUnreadPlayerBytes(t *testing.T) {
	in := copyFixture(t, "Steam_333333_tugen.fch")
	data := readFile(t, in)
	data = insertPayloadBytes(data, len(data)-68, []byte{0xde, 0xad, 0xbe, 0xef})
	writeTestFile(t, in, data)

	err := run([]string{"--character", in, "--no-backup", "set", "player-stat", "Deaths", "1"}, ioDiscard{}, ioDiscard{})
	if err == nil || !strings.Contains(err.Error(), "unread player bytes") {
		t.Fatalf("run error = %v, want unread player bytes", err)
	}
	if got := readFile(t, in); !bytes.Equal(got, data) {
		t.Fatal("fchedit modified a file with unread player bytes")
	}
}

func copyFixture(t *testing.T, name string) string {
	t.Helper()
	data := readFile(t, filepath.Join("..", "..", "testdata", name))
	path := filepath.Join(t.TempDir(), name)
	writeTestFile(t, path, data)
	return path
}

func decodeFile(t *testing.T, path string) *fch.Character {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	got, err := fch.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func insertPayloadBytes(data []byte, offset int, inserted []byte) []byte {
	out := make([]byte, 0, len(data)+len(inserted))
	out = append(out, data[:offset]...)
	out = append(out, inserted...)
	out = append(out, data[offset:]...)

	payloadLen := binary.LittleEndian.Uint32(out[:4]) + uint32(len(inserted))
	binary.LittleEndian.PutUint32(out[:4], payloadLen)
	hash := sha512.Sum512(out[4 : 4+payloadLen])
	copy(out[8+payloadLen:], hash[:])
	return out
}

func findItem(items []fch.Item, name string) *fch.Item {
	for i := range items {
		if items[i].Name == name {
			return &items[i]
		}
	}
	return nil
}

func findSlot(items []fch.Item, gridX int32, gridY int32) *fch.Item {
	for i := range items {
		if items[i].GridX == gridX && items[i].GridY == gridY {
			return &items[i]
		}
	}
	return nil
}

func nextEmptyInventorySlot(items []fch.Item) (int32, int32, bool) {
	for y := int32(0); y < 4; y++ {
		for x := int32(0); x < 8; x++ {
			if findSlot(items, x, y) == nil {
				return x, y, true
			}
		}
	}
	return 0, 0, false
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

func requireFcheditLastModified(t *testing.T, character *fch.Character) string {
	t.Helper()
	value := requireCustomDataValue(t, character, fcheditLastModifiedKey)
	requireTimestamp(t, fcheditLastModifiedKey, value, fcheditLastModifiedValue)
	return value
}

func requireCustomDataValue(t *testing.T, character *fch.Character, key string) string {
	t.Helper()
	value, ok := customDataValue(character.Player.CustomData, key)
	if !ok {
		t.Fatalf("missing %s custom data", key)
	}
	return value
}

func requireTimestamp(t *testing.T, name string, value string, layout string) {
	t.Helper()
	if _, err := time.Parse(layout, value); err != nil {
		t.Fatalf("%s = %q, want %s timestamp: %v", name, value, layout, err)
	}
}

func customDataValue(entries []fch.TextEntry, key string) (string, bool) {
	for _, entry := range entries {
		if entry.Key == key {
			return entry.Value, true
		}
	}
	return "", false
}

func countCustomData(entries []fch.TextEntry, key string) int {
	count := 0
	for _, entry := range entries {
		if entry.Key == key {
			count++
		}
	}
	return count
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
