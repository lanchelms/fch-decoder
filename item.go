package fch

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

func (i *Item) Decode(r *Reader) {
	i.Name = r.str()
	i.Stack = r.i32()
	i.Durability = r.f32()
	i.GridX = r.i32()
	i.GridY = r.i32()
	i.Equipped = r.bool()
	i.Quality = r.i32()
	i.Variant = r.i32()
	i.CrafterID = r.u64()
	i.CrafterName = r.str()
	i.CustomData = readList[TextEntry](r)
	i.WorldLevel = r.u32()
	i.PickedUp = r.bool()
}

func (i Item) Encode(w *Writer) {
	w.str(i.Name)
	w.i32(i.Stack)
	w.f32(i.Durability)
	w.i32(i.GridX)
	w.i32(i.GridY)
	w.bool(i.Equipped)
	w.i32(i.Quality)
	w.i32(i.Variant)
	w.u64(i.CrafterID)
	w.str(i.CrafterName)
	writeList(w, i.CustomData)
	w.u32(i.WorldLevel)
	w.bool(i.PickedUp)
}
