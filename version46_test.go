package fch

import (
	"bytes"
	"encoding/binary"
	"os"
	"reflect"
	"testing"

	wire "github.com/lanchelms/fch-decoder/binary"
	"github.com/lanchelms/fch-decoder/valheim"
)

const fixture46 = "testdata/Steam_444444_nichael.fch"

func TestVersion46Sample(t *testing.T) {
	c, err := DecodeFile(fixture46)
	if err != nil {
		t.Fatal(err)
	}
	if c.Version != 46 || c.Player.PlayerVersion != 33 || c.Player.InventoryVersion != 109 || c.Player.SkillVersion != 2 {
		t.Fatal("unexpected source versions")
	}
	if !c.Trailer.HashValid || c.RemainingBytes != 0 || c.PlayerStatCount != 205 || len(c.StatGroups) != 10 {
		t.Fatal("invalid character shape")
	}
	names := []string{"RawStats", "Any", "Hammer", "Casual", "VeryEasy", "Easy", "Default", "Hard", "VeryHard", "Hardcore"}
	for i, g := range c.StatGroups {
		if g.Name != names[i] || len(g.Stats) != 205 || len(g.EnemyStats) != 5 {
			t.Fatalf("bad group %d", i)
		}
	}
	if c.StatGroups[0].Stats[0].Value != 34 || c.StatGroups[0].Stats[2].Value != 29793 || c.StatGroups[6].Stats[0].Value != 0 {
		t.Fatal("incorrect saved statistics")
	}
	if len(c.Player.Inventory) != 35 || c.Player.Inventory[0].Name != "BeltStrength" || c.Player.Inventory[0].Durability != 100 || !c.Player.Inventory[0].Equipped {
		t.Fatal("incorrect inventory")
	}
	if c.Player.PlayerID != 444444 || c.Player.Name != "Nichael" || len(c.Player.KnownBiomeNames) != 14 || len(c.Player.BuildMenu) != 937 || c.Player.Stamina != float32(180.5907) {
		t.Fatalf("incorrect player tail: blob %d", len(c.Player.BuildMenu))
	}
	matchingCrafters := 0
	for _, item := range c.Player.Inventory {
		if item.CrafterID == 444444 {
			matchingCrafters++
		}
	}
	if matchingCrafters != 22 {
		t.Fatalf("matching crafter IDs = %d", matchingCrafters)
	}
	original, _ := os.ReadFile(fixture46)
	encoded, err := EncodeBytes(c)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, encoded) {
		t.Fatal("unedited round trip differs")
	}
}

func TestVersion46EditsPreserveOtherData(t *testing.T) {
	c, err := DecodeFile(fixture46)
	if err != nil {
		t.Fatal(err)
	}
	c.SetPlayerStat(0, "Deaths", 123)
	c.UpsertEnemyStat("test_enemy", 7)
	c.UpsertMaterialStat("test_item", 9)
	c.Player.Inventory[0].Durability = 12.34
	data, err := EncodeBytes(c)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.StatGroups[0].Stats[0].Value != 123 {
		t.Fatal("stat edit missing")
	}
	if v, _ := got.EnemyStat("test_enemy"); v != 7 {
		t.Fatal("enemy edit missing")
	}
	if v, _ := got.MaterialStat("test_item"); v != 9 {
		t.Fatal("item edit missing")
	}
	if !reflect.DeepEqual(c.StatGroups[1:], got.StatGroups[1:]) || !bytes.Equal(c.Player.BuildMenu, got.Player.BuildMenu) || !bytes.Equal(c.Map.Raw, got.Map.Raw) {
		t.Fatal("unrelated data changed")
	}
	if got.Player.Inventory[0].Durability != float32(1234)*0.01 {
		t.Fatal("durability edit missing")
	}
}

func TestVersion46MalformedPayload(t *testing.T) {
	original, err := os.ReadFile(fixture46)
	if err != nil {
		t.Fatal(err)
	}
	c, err := DecodeBytes(original)
	if err != nil {
		t.Fatal(err)
	}
	playerStart := len(original) - 68 - int(c.PlayerDataLength)
	for name, offset := range map[string]int{"character version": 4, "stat count": 8, "group count": 12, "world history count": 16 + 205*4, "player length": playerStart - 4, "player version": playerStart} {
		t.Run(name, func(t *testing.T) {
			b := bytes.Clone(original)
			binary.LittleEndian.PutUint32(b[offset:], 0x7fffffff)
			if _, err := DecodeBytes(b); err == nil {
				t.Fatal("accepted invalid field")
			}
		})
	}
	// Truncate inside every major section while retaining a consistent envelope.
	for _, end := range []int{16, 100, 836, playerStart + 3, playerStart + 24, len(original) - 73} {
		b := append(bytes.Clone(original[:end]), make([]byte, 68)...)
		binary.LittleEndian.PutUint32(b, uint32(end-4))
		binary.LittleEndian.PutUint32(b[end:], 64)
		if _, err := DecodeBytes(b); err == nil {
			t.Fatalf("accepted truncated payload at %d", end)
		}
	}
}

func TestStructuralMaps(t *testing.T) {
	for _, count := range []uint32{0, 1, 3} {
		t.Run(string(rune('0'+count)), func(t *testing.T) {
			c := valheim.NewCharacter("Map", 1)
			w := wire.NewWriter()
			w.Bool(true)
			w.Uint32(count)
			for j := uint32(0); j < count; j++ {
				w.Bytes(make([]byte, 59))
				w.Bool(j%2 == 0)
				if j%2 == 0 {
					w.Uint32(4)
					w.Bytes([]byte{1, 2, 3, 4})
				}
			}
			c.Map.Raw = w.Data()
			data, err := EncodeBytes(c)
			if err != nil {
				t.Fatal(err)
			}
			got, err := DecodeBytes(data)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(c.Map.Raw, got.Map.Raw) {
				t.Fatal("map changed")
			}
		})
	}
}

func TestVersion46PlayerLengthBounds(t *testing.T) {
	original, err := os.ReadFile(fixture46)
	if err != nil {
		t.Fatal(err)
	}
	c, err := DecodeBytes(original)
	if err != nil {
		t.Fatal(err)
	}
	offset := len(original) - 68 - int(c.PlayerDataLength) - 4
	for _, delta := range []int{-1, 1} {
		b := bytes.Clone(original)
		binary.LittleEndian.PutUint32(b[offset:], uint32(int(c.PlayerDataLength)+delta))
		if _, err := DecodeBytes(b); err == nil {
			t.Fatalf("accepted player length delta %d", delta)
		}
	}
}

func TestMalformedMapRecords(t *testing.T) {
	for _, badLength := range []bool{false, true} {
		c := valheim.NewCharacter("Map", 1)
		w := wire.NewWriter()
		w.Bool(true)
		if badLength {
			w.Uint32(1)
			w.Bytes(make([]byte, 59))
			w.Bool(true)
			w.Uint32(0xffffffff)
		} else {
			w.Uint32(0xffffffff)
		}
		w.Bytes([]byte{0x1f, 0x8b, 0x08})
		c.Map.Raw = w.Data()
		data, err := EncodeBytes(c)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeBytes(data); err == nil {
			t.Fatal("accepted malformed structural map")
		}
	}
}

func TestVersion46UnsupportedNestedVersions(t *testing.T) {
	original, err := os.ReadFile(fixture46)
	if err != nil {
		t.Fatal(err)
	}
	c, err := DecodeBytes(original)
	if err != nil {
		t.Fatal(err)
	}
	playerStart := len(original) - 68 - int(c.PlayerDataLength)
	r := wire.NewReader(original)
	r.SetPosition(playerStart + 20)
	r.String()
	r.Float32()
	inventoryOffset := r.Position()
	for _, version := range []uint32{0, 107, 108, 110} {
		b := bytes.Clone(original)
		binary.LittleEndian.PutUint32(b[inventoryOffset:], version)
		if _, err := DecodeBytes(b); err == nil {
			t.Fatalf("accepted inventory version %d", version)
		}
	}
}
