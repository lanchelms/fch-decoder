package valheim

import "github.com/lanchelms/fch-decoder/binary"

type Biome uint32

func (b *Biome) Decode(r *binary.Reader) {
	*b = Biome(r.Uint32())
}

func (b Biome) Encode(w *binary.Writer) {
	w.Uint32(uint32(b))
}
