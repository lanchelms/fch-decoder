package valheim

import "testing"

func TestTimeEntryDecode(t *testing.T) {
	w := NewWriter()
	w.String("World")
	w.Float32(42)

	got := readValue[TimeEntry, *TimeEntry](NewReader(w.Data()))
	want := TimeEntry{Name: "World", Seconds: 42}
	if got != want {
		t.Fatalf("TimeEntry = %#v, want %#v", got, want)
	}
}
