package valheim

import "testing"

func TestWorldKeyDecodeRaw(t *testing.T) {
	w := NewWriter()
	w.String("nomap")
	w.Float32(7)

	got := readValue[WorldKey, *WorldKey](NewReader(w.Data()))
	want := WorldKey{Raw: "nomap", Seconds: 7}
	if got != want {
		t.Fatalf("WorldKey = %#v, want %#v", got, want)
	}
}

func TestWorldKeyDecodeSplit(t *testing.T) {
	w := NewWriter()
	w.String("PlayerDamage default")
	w.Float32(1375)

	got := readValue[WorldKey, *WorldKey](NewReader(w.Data()))
	want := WorldKey{Raw: "PlayerDamage default", Key: "PlayerDamage", Setting: "default", Seconds: 1375}
	if got != want {
		t.Fatalf("WorldKey = %#v, want %#v", got, want)
	}
}
