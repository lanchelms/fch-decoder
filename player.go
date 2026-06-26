package fch

import (
	"fmt"
	"time"
)

type Player struct {
	Name            string       `json:"name"`
	PlayerID        uint64       `json:"playerId"`
	StartSeed       string       `json:"startSeed"`
	UsedCheats      bool         `json:"usedCheats"`
	DateCreatedUnix int64        `json:"dateCreatedUnix"`
	KnownWorlds     []TimedEntry `json:"knownWorlds,omitempty"`
	KnownWorldKeys  []WorldKey   `json:"knownWorldKeys,omitempty"`
	KnownCommands   []StatEntry  `json:"-"`
	EnemyStats      []StatEntry  `json:"enemyStats,omitempty"`
	MaterialStats   []StatEntry  `json:"materialStats,omitempty"`
	RecipeStats     []StatEntry  `json:"recipeStats,omitempty"`
	PlayerState
	PlayerTail
}

func NewPlayer(name string, playerID uint64) Player {
	return Player{
		Name:            name,
		PlayerID:        playerID,
		DateCreatedUnix: time.Now().Unix(),
		PlayerState: PlayerState{
			PlayerVersion:    supportedPlayerVersion,
			InventoryVersion: supportedInventoryVersion,
		},
		PlayerTail: PlayerTail{
			SkillVersion: supportedSkillVersion,
		},
	}
}

func (p *Player) Decode(r *Reader) {
	p.Name = r.str()
	p.PlayerID = r.u64()
	p.StartSeed = r.str()
	p.UsedCheats = r.bool()
	p.DateCreatedUnix = int64(r.u64())

	p.KnownWorlds = readList[TimedEntry](r)
	p.KnownWorldKeys = readList[WorldKey](r)
	p.KnownCommands = readList[StatEntry](r)
	p.EnemyStats = readList[StatEntry](r)
	p.MaterialStats = readList[StatEntry](r)
	p.RecipeStats = readList[StatEntry](r)
}

func (p Player) Encode(w *Writer) {
	w.str(p.Name)
	w.u64(p.PlayerID)
	w.str(p.StartSeed)
	w.bool(p.UsedCheats)
	w.u64(uint64(p.DateCreatedUnix))

	writeList(w, p.KnownWorlds)
	writeList(w, p.KnownWorldKeys)
	writeList(w, p.KnownCommands)
	writeList(w, p.EnemyStats)
	writeList(w, p.MaterialStats)
	writeList(w, p.RecipeStats)
}

func (p Player) Validate() error {
	if err := p.PlayerState.Validate(); err != nil {
		return err
	}
	return p.PlayerTail.Validate()
}

type PlayerState struct {
	GuardianPower    GuardianPower `json:"guardianPower"`
	PlayerVersion    uint32        `json:"playerVersion"`
	MaxHealth        float32       `json:"maxHealth"`
	Health           float32       `json:"health"`
	MaxStamina       float32       `json:"maxStamina"`
	TimeSinceDeath   float32       `json:"timeSinceDeath"`
	InventoryVersion uint32        `json:"inventoryVersion"`
	Inventory        []Item        `json:"inventory,omitempty"`
}

func (s *PlayerState) Decode(r *Reader) {
	s.PlayerVersion = r.u32()
	s.MaxHealth = r.f32()
	s.Health = r.f32()
	s.MaxStamina = r.f32()
	s.TimeSinceDeath = r.f32()
	s.GuardianPower.Decode(r)
	s.InventoryVersion = r.u32()
	s.Inventory = readList[Item, *Item](r)
}

func (s PlayerState) Encode(w *Writer) {
	w.u32(s.PlayerVersion)
	w.f32(s.MaxHealth)
	w.f32(s.Health)
	w.f32(s.MaxStamina)
	w.f32(s.TimeSinceDeath)
	s.GuardianPower.Encode(w)
	w.u32(s.InventoryVersion)
	writeList(w, s.Inventory)
}

func (s PlayerState) Validate() error {
	if s.PlayerVersion != supportedPlayerVersion {
		return fmt.Errorf("unsupported player version %d", s.PlayerVersion)
	}
	if s.InventoryVersion != supportedInventoryVersion {
		return fmt.Errorf("unsupported inventory version %d", s.InventoryVersion)
	}
	return nil
}

type PlayerTail struct {
	KnownRecipes     []string    `json:"knownRecipes,omitempty"`
	KnownStations    []Station   `json:"knownStations,omitempty"`
	KnownMaterials   []string    `json:"knownMaterials,omitempty"`
	ShownTutorials   []string    `json:"-"`
	Uniques          []string    `json:"uniques,omitempty"`
	Trophies         []string    `json:"trophies,omitempty"`
	KnownBiomes      []Biome     `json:"knownBiomes,omitempty"`
	PlayerKnownTexts []TextEntry `json:"-"`
	Beard            string      `json:"beard,omitempty"`
	Hair             string      `json:"hair,omitempty"`
	SkinColor        Vector3     `json:"skinColor"`
	HairColor        Vector3     `json:"hairColor"`
	ModelIndex       uint32      `json:"modelIndex"`
	Foods            []Food      `json:"foods,omitempty"`
	SkillVersion     uint32      `json:"skillVersion,omitempty"`
	Skills           []Skill     `json:"skills,omitempty"`
	CustomData       []TextEntry `json:"customData,omitempty"`
	Stamina          float32     `json:"stamina"`
	MaxEitr          float32     `json:"maxEitr"`
	Eitr             float32     `json:"eitr"`
	tailFloatCount   int
}

func (t *PlayerTail) Decode(r *Reader) {
	t.KnownRecipes = readStringList(r)
	t.KnownStations = readList[Station](r)
	t.KnownMaterials = readStringList(r)
	t.ShownTutorials = readStringList(r)
	t.Uniques = readStringList(r)
	t.Trophies = readStringList(r)
	t.KnownBiomes = readList[Biome](r)
	t.PlayerKnownTexts = readList[TextEntry](r)

	t.Beard = r.str()
	t.Hair = r.str()
	t.SkinColor.Decode(r)
	t.HairColor.Decode(r)
	t.ModelIndex = r.u32()

	t.Foods = readList[Food](r)

	t.SkillVersion = r.u32()
	t.Skills = readList[Skill](r)
	t.CustomData = readList[TextEntry](r)

	if r.remaining() >= 8 {
		t.Stamina = r.f32()
		t.MaxEitr = r.f32()
		t.tailFloatCount = 2
	}
	if r.remaining() >= 4 {
		t.Eitr = r.f32()
		t.tailFloatCount = 3
	}
}

func (t PlayerTail) Encode(w *Writer) {
	writeStringList(w, t.KnownRecipes)
	writeList(w, t.KnownStations)
	writeStringList(w, t.KnownMaterials)
	writeStringList(w, t.ShownTutorials)
	writeStringList(w, t.Uniques)
	writeStringList(w, t.Trophies)
	writeList(w, t.KnownBiomes)
	writeList(w, t.PlayerKnownTexts)

	w.str(t.Beard)
	w.str(t.Hair)
	t.SkinColor.Encode(w)
	t.HairColor.Encode(w)
	w.u32(t.ModelIndex)

	writeList(w, t.Foods)

	w.u32(t.SkillVersion)
	writeList(w, t.Skills)
	writeList(w, t.CustomData)

	tailFloatCount := t.normalizedTailFloatCount()
	if tailFloatCount >= 2 {
		w.f32(t.Stamina)
		w.f32(t.MaxEitr)
	}
	if tailFloatCount >= 3 {
		w.f32(t.Eitr)
	}
}

func (t PlayerTail) Validate() error {
	if t.SkillVersion != supportedSkillVersion {
		return fmt.Errorf("unsupported skill version %d", t.SkillVersion)
	}
	tailFloatCount := t.normalizedTailFloatCount()
	if tailFloatCount != 2 && tailFloatCount != 3 {
		return fmt.Errorf("fch: unsupported player tail float count %d", tailFloatCount)
	}
	return nil
}

func (t PlayerTail) normalizedTailFloatCount() int {
	if t.tailFloatCount != 0 {
		return t.tailFloatCount
	}
	return 3
}

func NewPlayerStats() []StatEntry {
	stats := make([]StatEntry, len(playerStatNames))
	for i, name := range playerStatNames {
		stats[i].Name = name
	}
	return stats
}
