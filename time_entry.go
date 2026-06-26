package fch

type TimeEntry struct {
	Name    string  `json:"name"`
	Seconds float32 `json:"seconds"`
}

func (t *TimeEntry) Decode(r *Reader) {
	t.Name = r.str()
	t.Seconds = r.f32()
}

func (t TimeEntry) Encode(w *Writer) {
	w.str(t.Name)
	w.f32(t.Seconds)
}
