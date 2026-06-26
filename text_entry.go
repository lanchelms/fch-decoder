package fch

type TextEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (t *TextEntry) Decode(r *Reader) {
	t.Key = r.str()
	t.Value = r.str()
}

func (t TextEntry) Encode(w *Writer) {
	w.str(t.Key)
	w.str(t.Value)
}
