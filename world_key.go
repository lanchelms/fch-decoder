package fch

import "strings"

type WorldKey struct {
	Raw     string  `json:"raw"`
	Key     string  `json:"key,omitempty"`
	Setting string  `json:"setting,omitempty"`
	Seconds float32 `json:"seconds"`
}

func NewWorldKey(raw string, seconds float32) WorldKey {
	key, setting, ok := strings.Cut(raw, " ")
	if !ok {
		return WorldKey{Raw: raw, Seconds: seconds}
	}
	return WorldKey{Raw: raw, Key: key, Setting: setting, Seconds: seconds}
}

func (wk *WorldKey) Decode(r *Reader) {
	*wk = NewWorldKey(r.str(), r.f32())
}

func (wk WorldKey) Encode(w *Writer) {
	raw := wk.Raw
	if raw == "" {
		raw = wk.Key
		if wk.Setting != "" {
			raw += " " + wk.Setting
		}
	}
	w.str(raw)
	w.f32(wk.Seconds)
}
