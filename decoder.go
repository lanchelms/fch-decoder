package fch

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"
)

const (
	fileLengthSize = 4
	trailerSize    = 68
	fileOverhead   = fileLengthSize + trailerSize
)

type decoder interface {
	Decode(*Reader)
}

type MapSection struct {
	Offset           int    `json:"offset"`
	CompressedLength uint32 `json:"compressedLength"`
	StoredLength     uint32 `json:"storedLength"`
	Raw              []byte `json:"-"`
}

type Trailer struct {
	Offset    int    `json:"offset"`
	Length    uint32 `json:"length"`
	Hash      []byte `json:"hash"`
	HashValid bool   `json:"hashValid"`
}

type StatEntry struct {
	Name  string  `json:"name"`
	Value float32 `json:"value"`
}

func (s *StatEntry) Decode(r *Reader) {
	s.Name = r.str()
	s.Value = r.f32()
}

func (s StatEntry) Encode(w *Writer) {
	w.str(s.Name)
	w.f32(s.Value)
}

type TimedEntry struct {
	Name    string  `json:"name"`
	Seconds float32 `json:"seconds"`
}

func (t *TimedEntry) Decode(r *Reader) {
	t.Name = r.str()
	t.Seconds = r.f32()
}

func (t TimedEntry) Encode(w *Writer) {
	w.str(t.Name)
	w.f32(t.Seconds)
}

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

type Station struct {
	Name  string `json:"name"`
	Level uint32 `json:"level"`
}

func (s *Station) Decode(r *Reader) {
	s.Name = r.str()
	s.Level = r.u32()
}

func (s Station) Encode(w *Writer) {
	w.str(s.Name)
	w.u32(s.Level)
}

type Food struct {
	Name string  `json:"name"`
	Time float32 `json:"time"`
}

func (f *Food) Decode(r *Reader) {
	f.Name = r.str()
	f.Time = r.f32()
}

func (f Food) Encode(w *Writer) {
	w.str(f.Name)
	w.f32(f.Time)
}

type Biome uint32

func (b *Biome) Decode(r *Reader) {
	*b = Biome(r.u32())
}

func (b Biome) Encode(w *Writer) {
	w.u32(uint32(b))
}

type Vector3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

func (v *Vector3) Decode(r *Reader) {
	v.X = r.f32()
	v.Y = r.f32()
	v.Z = r.f32()
}

func (v Vector3) Encode(w *Writer) {
	w.f32(v.X)
	w.f32(v.Y)
	w.f32(v.Z)
}

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

func Decode(r io.Reader) (*Character, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return DecodeBytes(data)
}

func DecodeBytes(data []byte) (character *Character, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			character = nil
			err = fmt.Errorf("fch: decode failed: %v", recovered)
		}
	}()

	if len(data) < fileOverhead+16 {
		return nil, fmt.Errorf("fch: file too short: %d bytes", len(data))
	}

	rd := NewReader(data)
	c := &Character{}
	c.FileLength = rd.u32()
	c.Version = rd.u32()
	c.PlayerStatCount = rd.u32()
	if int(c.FileLength)+fileOverhead != len(data) {
		return nil, fmt.Errorf("fch: length header %d does not match file size %d", c.FileLength, len(data))
	}

	rd = NewReader(data)
	c.Decode(rd)
	return c, nil
}

func currentPayloadHash(data []byte, payloadLen uint32) []byte {
	return payloadHash(data[fileLengthSize : fileLengthSize+int(payloadLen)])
}

func readMapSection(data []byte, startOffset int, payloadEnd int) (MapSection, int, error) {
	firstSpawn, worldCount, ok := readMapPrefix(data, startOffset, payloadEnd)
	if ok && firstSpawn <= 1 && worldCount == 0 {
		return MapSection{
			Offset: startOffset,
			Raw:    append([]byte(nil), data[startOffset:startOffset+5]...),
		}, startOffset + 5, nil
	}

	gzipOffset := bytes.Index(data[startOffset:], []byte{0x1f, 0x8b, 0x08})
	if gzipOffset < 0 {
		return MapSection{}, 0, fmt.Errorf("fch: gzip map block not found")
	}
	gzipOffset += startOffset
	if gzipOffset < 12 {
		return MapSection{}, 0, fmt.Errorf("fch: gzip map block starts too early")
	}

	storedLen := binary.LittleEndian.Uint32(data[gzipOffset-12 : gzipOffset-8])
	compressedLen := binary.LittleEndian.Uint32(data[gzipOffset-4 : gzipOffset])
	if gzipOffset+int(compressedLen) > payloadEnd {
		return MapSection{}, 0, fmt.Errorf("fch: invalid compressed map length %d at offset %d", compressedLen, gzipOffset)
	}

	return MapSection{
		Offset:           gzipOffset,
		CompressedLength: compressedLen,
		StoredLength:     storedLen,
		Raw:              append([]byte(nil), data[startOffset:gzipOffset+int(compressedLen)]...),
	}, gzipOffset + int(compressedLen), nil
}

func readMapPrefix(data []byte, startOffset int, payloadEnd int) (byte, uint32, bool) {
	if startOffset+5 > payloadEnd {
		return 0, 0, false
	}
	firstSpawn := data[startOffset]
	worldCount := binary.LittleEndian.Uint32(data[startOffset+1 : startOffset+5])
	return firstSpawn, worldCount, true
}
