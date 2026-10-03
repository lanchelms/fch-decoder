package valheim

import (
	"fmt"
	"math"
	"sync"
	"unicode/utf16"

	"github.com/lanchelms/fch-decoder/binary"
	"github.com/lanchelms/fch-decoder/valheim/items"
)

var prefabNames = sync.OnceValue(func() map[int32]string {
	return indexPrefabNames(items.Catalog().Names())
})

func (i *Item) decodeCompact(r *binary.Reader) {
	i.compactDecoded = true
	i.compactDurability = r.Int32()
	i.Durability = float32(i.compactDurability) * 0.01
	i.GridX = int32(r.Byte())
	i.GridY = int32(r.Byte())
	i.WorldLevel = uint32(r.Byte())
	f := r.Byte()
	i.compactFlags = f
	i.PickedUp = f&1 != 0
	i.Equipped = f&2 != 0
	i.Quality = 1
	if f&4 != 0 {
		i.Quality = int32(r.Uint16())
	}
	i.Stack = 1
	if f&8 != 0 {
		i.Stack = int32(r.Uint16())
	}
	if f&16 != 0 {
		i.Variant = r.Int32()
	}
	if f&32 != 0 {
		i.CrafterID = r.Uint64()
		i.CrafterName = r.String()
	}
	if f&64 != 0 {
		i.PrefabHash = r.Int32()
		i.Name = prefabNames()[i.PrefabHash]
	}
	if f&128 != 0 {
		start := r.Position()
		n := r.NumItems()
		i.compactWideCount = r.Position()-start == 2
		for range n {
			i.CustomData = append(i.CustomData, readValue[TextEntry](r))
		}
	}
	i.compactExtra = r.Byte()
	i.Cheated = i.compactExtra&1 != 0
}

func (i Item) encodeCompact(w *binary.Writer) {
	durability := int32(i.Durability * 100)
	if i.compactDecoded && i.Durability == float32(i.compactDurability)*0.01 {
		durability = i.compactDurability
	}
	w.Int32(durability)
	w.Byte(byte(i.GridX))
	w.Byte(byte(i.GridY))
	w.Byte(byte(i.WorldLevel))
	f := i.encodingFlags()
	w.Byte(f)
	if f&4 != 0 {
		w.Uint16(uint16(i.Quality))
	}
	if f&8 != 0 {
		w.Uint16(uint16(i.Stack))
	}
	if f&16 != 0 {
		w.Int32(i.Variant)
	}
	if f&32 != 0 {
		w.Uint64(i.CrafterID)
		w.String(i.CrafterName)
	}
	if f&64 != 0 {
		h := i.PrefabHash
		if i.Name != "" {
			h = prefabHash(i.Name)
		}
		w.Int32(h)
	}
	if f&128 != 0 {
		n := len(i.CustomData)
		if i.compactWideCount && n < 128 {
			w.Byte(128)
			w.Byte(byte(n))
		} else {
			w.NumItems(n)
		}
		for _, entry := range i.CustomData {
			entry.Encode(w)
		}
	}
	extra := i.compactExtra &^ 1
	if i.Cheated {
		extra |= 1
	}
	w.Byte(extra)
}

// encodingFlags retains explicit optional defaults from decoded records.
func (i Item) encodingFlags() byte {
	f := i.compactFlags & 252
	if i.PickedUp {
		f |= 1
	}
	if i.Equipped {
		f |= 2
	}
	if i.Quality != 1 {
		f |= 4
	}
	if i.Stack != 1 {
		f |= 8
	}
	if i.Variant != 0 {
		f |= 16
	}
	if i.CrafterID != 0 || i.CrafterName != "" {
		f |= 32
	}
	if i.Name != "" || i.PrefabHash != 0 {
		f |= 64
	}
	if len(i.CustomData) > 0 {
		f |= 128
	}
	return f
}

func (i Item) validateCompact() error {
	if i.GridX < 0 || i.GridX > 255 || i.GridY < 0 || i.GridY > 255 || i.WorldLevel > 255 || i.Quality < 0 || i.Quality > 65535 || i.Stack < 0 || i.Stack > 65535 || len(i.CustomData) > 32767 {
		return fmt.Errorf("item %q exceeds compact field limits", i.Name)
	}
	if i.compactDecoded && i.Durability == float32(i.compactDurability)*0.01 {
		return nil
	}
	d := float64(i.Durability * 100)
	if math.IsNaN(d) || math.IsInf(d, 0) || d < math.MinInt32 || d > math.MaxInt32 {
		return fmt.Errorf("item %q durability exceeds compact limits", i.Name)
	}
	return nil
}

func prefabHash(name string) int32 {
	chars := utf16.Encode([]rune(name))
	a, b := int32(5381), int32(5381)
	for j := 0; j < len(chars) && chars[j] != 0; j += 2 {
		a = a*33 ^ int32(chars[j])
		if j+1 == len(chars) || chars[j+1] == 0 {
			break
		}
		b = b*33 ^ int32(chars[j+1])
	}
	return a + b*1566083941
}

func indexPrefabNames(prefabs []string) map[int32]string {
	names := map[int32]string{}
	for _, name := range prefabs {
		h := prefabHash(name)
		if _, ok := names[h]; ok {
			names[h] = ""
		} else {
			names[h] = name
		}
	}
	return names
}
