package fch

import "math"

type Skill struct {
	Type         int32   `json:"type"`
	Name         string  `json:"name,omitempty"`
	Level        float32 `json:"level"`
	DisplayLevel int32   `json:"displayLevel"`
	Accumulator  float32 `json:"accumulator"`
}

func (s *Skill) Decode(r *Reader) {
	s.Type = r.i32()
	s.Name = skillName(s.Type)
	s.Level = r.f32()
	s.DisplayLevel = s.displayLevel()
	s.Accumulator = r.f32()
}

func (s Skill) Encode(w *Writer) {
	w.i32(s.Type)
	w.f32(s.Level)
	w.f32(s.Accumulator)
}

func (s Skill) displayLevel() int32 {
	return int32(math.Floor(float64(s.Level)))
}
