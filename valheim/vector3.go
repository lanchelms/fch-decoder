package valheim

import "github.com/lanchelms/fch-decoder/binary"

type Vector3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

func (v *Vector3) Decode(r *binary.Reader) {
	v.X = r.Float32()
	v.Y = r.Float32()
	v.Z = r.Float32()
}

func (v Vector3) Encode(w *binary.Writer) {
	w.Float32(v.X)
	w.Float32(v.Y)
	w.Float32(v.Z)
}
