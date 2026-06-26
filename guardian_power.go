package fch

type GuardianPower struct {
	Name     string  `json:"name"`
	Cooldown float32 `json:"cooldown"`
}

func (g *GuardianPower) Decode(r *Reader) {
	g.Name = r.str()
	g.Cooldown = r.f32()
}

func (g GuardianPower) Encode(w *Writer) {
	w.str(g.Name)
	w.f32(g.Cooldown)
}
