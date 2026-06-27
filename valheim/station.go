package valheim

import "github.com/lanchelms/fch-decoder/binary"

type Station struct {
	Name  string `json:"name"`
	Level uint32 `json:"level"`
}

func (s *Station) Decode(r *binary.Reader) {
	s.Name = r.String()
	s.Level = r.Uint32()
}

func (s Station) Encode(w *binary.Writer) {
	w.String(s.Name)
	w.Uint32(s.Level)
}
