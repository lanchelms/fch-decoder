package valheim

import "github.com/lanchelms/fch-decoder/binary"

type TextEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (t *TextEntry) Decode(r *binary.Reader) {
	t.Key = r.String()
	t.Value = r.String()
}

func (t TextEntry) Encode(w *binary.Writer) {
	w.String(t.Key)
	w.String(t.Value)
}
