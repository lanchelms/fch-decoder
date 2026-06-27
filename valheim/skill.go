package valheim

import (
	"math"

	"github.com/lanchelms/fch-decoder/binary"
)

type Skill struct {
	Type         int32   `json:"type"`
	Name         string  `json:"name,omitempty"`
	Level        float32 `json:"level"`
	DisplayLevel int32   `json:"displayLevel"`
	Accumulator  float32 `json:"accumulator"`
}

func (s *Skill) Decode(r *binary.Reader) {
	s.Type = r.Int32()
	s.Name = skillName(s.Type)
	s.Level = r.Float32()
	s.DisplayLevel = s.displayLevel()
	s.Accumulator = r.Float32()
}

func (s Skill) Encode(w *binary.Writer) {
	w.Int32(s.Type)
	w.Float32(s.Level)
	w.Float32(s.Accumulator)
}

func (s Skill) displayLevel() int32 {
	return int32(math.Floor(float64(s.Level)))
}
