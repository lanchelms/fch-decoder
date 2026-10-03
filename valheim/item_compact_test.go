package valheim

import (
	"bytes"
	"math"
	"testing"

	"github.com/lanchelms/fch-decoder/binary"
)

func TestCompactItemWirePreservation(t *testing.T) {
	for _, durability := range []int32{0, 1, 1234, 16777217, math.MaxInt32, math.MinInt32} {
		w := binary.NewWriter()
		w.Int32(durability)
		w.Bytes([]byte{255, 254, 253, 255})
		w.Uint16(1)
		w.Uint16(1)
		w.Int32(0)
		w.Uint64(0)
		w.String("")
		w.Int32(123456789)
		w.Bytes([]byte{128, 0, 129})
		var i Item
		i.decodeCompact(binary.NewReader(w.Data()))
		if i.Name != "" || i.PrefabHash != 123456789 || !i.Cheated || !i.PickedUp || !i.Equipped {
			t.Fatalf("incorrect item: %+v", i)
		}
		if err := i.validateCompact(); err != nil {
			t.Fatal(err)
		}
		encoded := binary.NewWriter()
		i.encodeCompact(encoded)
		if !bytes.Equal(w.Data(), encoded.Data()) {
			t.Fatalf("wire data changed for durability %d", durability)
		}
	}
}

func TestCompactItemDefaultsAndLimits(t *testing.T) {
	w := binary.NewWriter()
	w.Bytes(make([]byte, 9))
	var i Item
	i.decodeCompact(binary.NewReader(w.Data()))
	if i.Quality != 1 || i.Stack != 1 || i.Variant != 0 || i.CrafterID != 0 || i.Name != "" || i.Cheated {
		t.Fatalf("bad defaults: %+v", i)
	}
	valid := Item{Name: "Wood", Quality: 65535, Stack: 65535, GridX: 255, GridY: 255, WorldLevel: 255, Durability: 12.34, Cheated: true, CustomData: []TextEntry{{Key: "test", Value: "value"}}}
	if err := valid.validateCompact(); err != nil {
		t.Fatal(err)
	}
	out := binary.NewWriter()
	valid.encodeCompact(out)
	var got Item
	got.decodeCompact(binary.NewReader(out.Data()))
	if got.Name != "Wood" || got.Durability != float32(1234)*0.01 || got.Stack != 65535 || got.CustomData[0] != valid.CustomData[0] || !got.Cheated {
		t.Fatalf("bad compact round trip: %+v", got)
	}
	for _, modify := range []func(*Item){
		func(i *Item) { i.GridX = -1 }, func(i *Item) { i.GridY = 256 }, func(i *Item) { i.WorldLevel = 256 }, func(i *Item) { i.Stack = 65536 }, func(i *Item) { i.Quality = -1 }, func(i *Item) { i.Durability = float32(math.NaN()) }, func(i *Item) { i.Durability = 21474836 }, func(i *Item) { i.CustomData = make([]TextEntry, 32768) },
	} {
		i := valid
		modify(&i)
		if err := i.validateCompact(); err == nil {
			t.Fatalf("accepted invalid item: %+v", i)
		}
	}
}

func TestCompactItemTruncation(t *testing.T) {
	w := binary.NewWriter()
	Item{Name: "Wood", Stack: 3, Quality: 2, CrafterID: 42, CrafterName: "Bob", CustomData: []TextEntry{{Key: "a", Value: "b"}}}.encodeCompact(w)
	data := w.Data()
	for n := 0; n < len(data); n++ {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("accepted truncated item at %d", n)
				}
			}()
			var i Item
			i.decodeCompact(binary.NewReader(data[:n]))
		}()
	}
}

func TestCompactCountBoundaries(t *testing.T) {
	for _, n := range []int{0, 127, 128, 255, 256, 32767} {
		w := binary.NewWriter()
		w.NumItems(n)
		r := binary.NewReader(w.Data())
		if got := r.NumItems(); got != n || r.Remaining() != 0 {
			t.Fatalf("count %d became %d", n, got)
		}
	}
	for _, n := range []int{-1, 32768} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("accepted count %d", n)
				}
			}()
			binary.NewWriter().NumItems(n)
		}()
	}
}

func TestPrefabHashCollision(t *testing.T) {
	const first, second = "bkZYvRsccLft", "gYqJMyZlRROC"
	if prefabHash(first) != prefabHash(second) {
		t.Fatal("collision fixture is invalid")
	}
	names := indexPrefabNames([]string{"Wood", first, second})
	if names[prefabHash("Wood")] != "Wood" || names[prefabHash(first)] != "" {
		t.Fatal("ambiguous prefab hash was resolved")
	}
}
