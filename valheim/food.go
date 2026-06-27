package valheim

import "github.com/lanchelms/fch-decoder/binary"

type Food struct {
	Name string  `json:"name"`
	Time float32 `json:"time"`
}

func (f *Food) Decode(r *binary.Reader) {
	f.Name = r.String()
	f.Time = r.Float32()
}

func (f Food) Encode(w *binary.Writer) {
	w.String(f.Name)
	w.Float32(f.Time)
}
