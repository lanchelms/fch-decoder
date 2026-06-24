package fch

import (
	"fmt"
	"io"
	"math"
)

func Encode(w io.Writer, c *Character) error {
	data, err := EncodeBytes(c)
	if err != nil {
		return err
	}
	n, err := w.Write(data)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}

func EncodeBytes(c *Character) ([]byte, error) {
	return newEncoder().character(c)
}

type encoder struct {
	w *writer
}

func newEncoder() *encoder {
	return &encoder{w: newWriter()}
}

func (e *encoder) character(c *Character) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("fch: cannot encode nil character")
	}
	if len(c.Map.Raw) == 0 {
		return nil, fmt.Errorf("fch: cannot encode character without raw map section")
	}

	payload := newEncoder()
	if err := payload.payload(c); err != nil {
		return nil, err
	}
	payloadBytes := payload.w.data()
	if len(payloadBytes) > math.MaxUint32 {
		return nil, fmt.Errorf("fch: payload too large: %d bytes", len(payloadBytes))
	}

	e.w.u32(uint32(len(payloadBytes)))
	e.w.bytes(payloadBytes)
	e.w.u32(payloadHashSize)
	e.w.bytes(payloadHash(payloadBytes))
	return e.w.data(), nil
}

func (e *encoder) payload(c *Character) error {
	if c.PlayerStatCount != 0 && int(c.PlayerStatCount) != len(c.PlayerStats) {
		return fmt.Errorf("fch: player stat count %d does not match %d stats", c.PlayerStatCount, len(c.PlayerStats))
	}

	e.w.u32(c.Version)
	e.w.u32(uint32(len(c.PlayerStats)))
	for _, stat := range c.PlayerStats {
		e.w.f32(stat.Value)
	}
	e.w.bytes(c.Map.Raw)
	return e.player(c.Player)
}

func (e *encoder) player(p PlayerData) error {
	e.w.str(p.Name)
	e.w.u64(p.PlayerID)
	e.w.str(p.StartSeed)
	e.w.bool(p.UsedCheats)
	e.w.u64(uint64(p.DateCreatedUnix))

	writeList(e.w, p.KnownWorlds, e.timedEntry)
	writeList(e.w, p.KnownWorldKeys, e.worldKey)
	writeList(e.w, p.KnownCommands, e.statEntry)
	writeList(e.w, p.EnemyStats, e.statEntry)
	writeList(e.w, p.MaterialStats, e.statEntry)
	writeList(e.w, p.RecipeStats, e.statEntry)
	e.playerState(p)
	e.inventory(p.Inventory)
	return e.playerTail(p)
}

func (e *encoder) playerState(p PlayerData) {
	e.w.bool(p.HasPlayerData)
	e.w.u32(p.PlayerDataLength)
	e.w.u32(p.PlayerVersion)
	e.w.f32(p.MaxHealth)
	e.w.f32(p.Health)
	e.w.f32(p.MaxStamina)
	e.w.f32(p.TimeSinceDeath)
	e.w.str(p.GuardianPower.Name)
	e.w.f32(p.GuardianPower.Cooldown)
	e.w.u32(p.InventoryVersion)
}

func (e *encoder) inventory(items []Item) {
	writeList(e.w, items, e.item)
}

func (e *encoder) playerTail(p PlayerData) error {
	writeList(e.w, p.KnownRecipes, e.str)
	writeList(e.w, p.KnownStations, e.station)
	writeList(e.w, p.KnownMaterials, e.str)
	writeList(e.w, p.ShownTutorials, e.str)
	writeList(e.w, p.Uniques, e.str)
	writeList(e.w, p.Trophies, e.str)
	writeList(e.w, p.KnownBiomes, e.biome)
	writeList(e.w, p.PlayerKnownTexts, e.textEntry)

	e.w.str(p.Beard)
	e.w.str(p.Hair)
	e.w.vector3(p.SkinColor)
	e.w.vector3(p.HairColor)
	e.w.u32(p.ModelIndex)

	writeList(e.w, p.Foods, e.food)

	e.w.u32(p.SkillVersion)
	writeList(e.w, p.Skills, e.skill)
	writeList(e.w, p.CustomData, e.textEntry)

	tailFloatCount := p.tailFloatCount
	if tailFloatCount == 0 {
		tailFloatCount = 3
	}
	switch tailFloatCount {
	case 2:
		e.w.f32(p.Stamina)
		e.w.f32(p.MaxEitr)
	case 3:
		e.w.f32(p.Stamina)
		e.w.f32(p.MaxEitr)
		e.w.f32(p.Eitr)
	default:
		return fmt.Errorf("fch: unsupported player tail float count %d", tailFloatCount)
	}
	return nil
}

func (e *encoder) str(v string) {
	e.w.str(v)
}

func (e *encoder) statEntry(v StatEntry) {
	e.w.str(v.Name)
	e.w.f32(v.Value)
}

func (e *encoder) timedEntry(v TimedEntry) {
	e.w.str(v.Name)
	e.w.f32(v.Seconds)
}

func (e *encoder) textEntry(v TextEntry) {
	e.w.str(v.Key)
	e.w.str(v.Value)
}

func (e *encoder) station(v Station) {
	e.w.str(v.Name)
	e.w.u32(v.Level)
}

func (e *encoder) biome(v uint32) {
	e.w.u32(v)
}

func (e *encoder) food(v Food) {
	e.w.str(v.Name)
	e.w.f32(v.Time)
}

func (e *encoder) skill(v Skill) {
	e.w.i32(v.Type)
	e.w.f32(v.Level)
	e.w.f32(v.Accumulator)
}

func (e *encoder) item(v Item) {
	e.w.str(v.Name)
	e.w.i32(v.Stack)
	e.w.f32(v.Durability)
	e.w.i32(v.GridX)
	e.w.i32(v.GridY)
	e.w.bool(v.Equipped)
	e.w.i32(v.Quality)
	e.w.i32(v.Variant)
	e.w.u64(v.CrafterID)
	e.w.str(v.CrafterName)
	writeList(e.w, v.CustomData, e.textEntry)
	e.w.u32(v.WorldLevel)
	e.w.bool(v.PickedUp)
}

func (e *encoder) worldKey(v WorldKey) {
	raw := v.Raw
	if raw == "" {
		raw = v.Key
		if v.Setting != "" {
			raw += " " + v.Setting
		}
	}
	e.w.str(raw)
	e.w.f32(v.Seconds)
}
