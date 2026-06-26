package fch

type Vector3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

func (v *Vector3) Decode(r *Reader) {
	v.X = r.f32()
	v.Y = r.f32()
	v.Z = r.f32()
}

func (v Vector3) Encode(w *Writer) {
	w.f32(v.X)
	w.f32(v.Y)
	w.f32(v.Z)
}
