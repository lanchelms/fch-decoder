package valheim

import "github.com/lanchelms/fch-decoder/binary"

type Item struct {
	Name        string      `json:"name"`
	Stack       int32       `json:"stack"`
	Durability  float32     `json:"durability"`
	GridX       int32       `json:"gridX"`
	GridY       int32       `json:"gridY"`
	Equipped    bool        `json:"equipped"`
	Quality     int32       `json:"quality"`
	Variant     int32       `json:"variant"`
	CrafterID   uint64      `json:"crafterId"`
	CrafterName string      `json:"crafterName"`
	CustomData  []TextEntry `json:"customData,omitempty"`
	WorldLevel  uint32      `json:"worldLevel"`
	PickedUp    bool        `json:"pickedUp"`
}

func (i *Item) Decode(r *binary.Reader) {
	i.Name = r.String()
	i.Stack = r.Int32()
	i.Durability = r.Float32()
	i.GridX = r.Int32()
	i.GridY = r.Int32()
	i.Equipped = r.Bool()
	i.Quality = r.Int32()
	i.Variant = r.Int32()
	i.CrafterID = r.Uint64()
	i.CrafterName = r.String()
	i.CustomData = readList[TextEntry](r)
	i.WorldLevel = r.Uint32()
	i.PickedUp = r.Bool()
}

func (i Item) Encode(w *binary.Writer) {
	w.String(i.Name)
	w.Int32(i.Stack)
	w.Float32(i.Durability)
	w.Int32(i.GridX)
	w.Int32(i.GridY)
	w.Bool(i.Equipped)
	w.Int32(i.Quality)
	w.Int32(i.Variant)
	w.Uint64(i.CrafterID)
	w.String(i.CrafterName)
	writeList(w, i.CustomData)
	w.Uint32(i.WorldLevel)
	w.Bool(i.PickedUp)
}
