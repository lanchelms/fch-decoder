package valheim

import "github.com/lanchelms/fch-decoder/binary"

type GuardianPower struct {
	Name     string  `json:"name"`
	Cooldown float32 `json:"cooldown"`
}

func (g *GuardianPower) Decode(r *binary.Reader) {
	g.Name = r.String()
	g.Cooldown = r.Float32()
}

func (g GuardianPower) Encode(w *binary.Writer) {
	w.String(g.Name)
	w.Float32(g.Cooldown)
}
