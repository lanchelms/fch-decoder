package valheim

import "github.com/lanchelms/fch-decoder/binary"

type StatEntry struct {
	Name  string  `json:"name"`
	Value float32 `json:"value"`
}

func (s *StatEntry) Decode(r *binary.Reader) {
	s.Name = r.String()
	s.Value = r.Float32()
}

func (s StatEntry) Encode(w *binary.Writer) {
	w.String(s.Name)
	w.Float32(s.Value)
}
