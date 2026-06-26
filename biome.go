package fch

type Biome uint32

func (b *Biome) Decode(r *Reader) {
	*b = Biome(r.u32())
}

func (b Biome) Encode(w *Writer) {
	w.u32(uint32(b))
}
