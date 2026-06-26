package fch

type Food struct {
	Name string  `json:"name"`
	Time float32 `json:"time"`
}

func (f *Food) Decode(r *Reader) {
	f.Name = r.str()
	f.Time = r.f32()
}

func (f Food) Encode(w *Writer) {
	w.str(f.Name)
	w.f32(f.Time)
}
