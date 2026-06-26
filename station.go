package fch

type Station struct {
	Name  string `json:"name"`
	Level uint32 `json:"level"`
}

func (s *Station) Decode(r *Reader) {
	s.Name = r.str()
	s.Level = r.u32()
}

func (s Station) Encode(w *Writer) {
	w.str(s.Name)
	w.u32(s.Level)
}
