package valheim

import "github.com/lanchelms/fch-decoder/binary"

type TimeEntry struct {
	Name    string  `json:"name"`
	Seconds float32 `json:"seconds"`
}

func (t *TimeEntry) Decode(r *binary.Reader) {
	t.Name = r.String()
	t.Seconds = r.Float32()
}

func (t TimeEntry) Encode(w *binary.Writer) {
	w.String(t.Name)
	w.Float32(t.Seconds)
}
