package fch

type StatEntry struct {
	Name  string  `json:"name"`
	Value float32 `json:"value"`
}

func (s *StatEntry) Decode(r *Reader) {
	s.Name = r.str()
	s.Value = r.f32()
}

func (s StatEntry) Encode(w *Writer) {
	w.str(s.Name)
	w.f32(s.Value)
}
