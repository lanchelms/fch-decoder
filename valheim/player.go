package valheim

import (
	"fmt"
	"time"

	"github.com/lanchelms/fch-decoder/binary"
)

type Player struct {
	Name            string      `json:"name"`
	PlayerID        uint64      `json:"playerId"`
	StartSeed       string      `json:"startSeed"`
	UsedCheats      bool        `json:"usedCheats"`
	DateCreatedUnix int64       `json:"dateCreatedUnix"`
	KnownWorlds     []TimeEntry `json:"knownWorlds,omitempty"`
	KnownWorldKeys  []WorldKey  `json:"knownWorldKeys,omitempty"`
	KnownCommands   []StatEntry `json:"-"`
	EnemyStats      []StatEntry `json:"enemyStats,omitempty"`
	MaterialStats   []StatEntry `json:"materialStats,omitempty"`
	RecipeStats     []StatEntry `json:"recipeStats,omitempty"`
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

func (p *Player) Decode(r *binary.Reader) {
	p.decodeVersion(r, 43)
}

func (p *Player) decodeVersion(r *binary.Reader, version uint32) {
	p.Name = r.String()
	p.PlayerID = r.Uint64()
	p.StartSeed = r.String()
	p.UsedCheats = r.Bool()
	p.DateCreatedUnix = int64(r.Uint64())

	if version == 46 {
		return
	}
	p.KnownWorlds = readList[TimeEntry](r)
	p.KnownWorldKeys = readList[WorldKey](r)
	p.KnownCommands = readList[StatEntry](r)
	p.EnemyStats = readList[StatEntry](r)
	p.MaterialStats = readList[StatEntry](r)
	p.RecipeStats = readList[StatEntry](r)
}

func (p Player) Encode(w *binary.Writer) {
	p.encodeVersion(w, 43)
}

func (p Player) encodeVersion(w *binary.Writer, version uint32) {
	w.String(p.Name)
	w.Uint64(p.PlayerID)
	w.String(p.StartSeed)
	w.Bool(p.UsedCheats)
	w.Uint64(uint64(p.DateCreatedUnix))

	if version == 46 {
		return
	}
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

func (s *PlayerState) Decode(r *binary.Reader) {
	s.PlayerVersion = r.Uint32()
	if s.PlayerVersion != 29 && s.PlayerVersion != 33 {
		panic(fmt.Errorf("unsupported player version %d", s.PlayerVersion))
	}
	s.MaxHealth = r.Float32()
	s.Health = r.Float32()
	s.MaxStamina = r.Float32()
	s.TimeSinceDeath = r.Float32()
	s.GuardianPower.Decode(r)
	s.InventoryVersion = r.Uint32()
	switch s.InventoryVersion {
	case 106:
		s.Inventory = readList[Item, *Item](r)
	case 109:
		n := r.Uint16()
		for range n {
			var item Item
			item.decodeCompact(r)
			s.Inventory = append(s.Inventory, item)
		}
	default:
		panic(fmt.Errorf("unsupported inventory version %d", s.InventoryVersion))
	}
}

func (s PlayerState) Encode(w *binary.Writer) {
	w.Uint32(s.PlayerVersion)
	w.Float32(s.MaxHealth)
	w.Float32(s.Health)
	w.Float32(s.MaxStamina)
	w.Float32(s.TimeSinceDeath)
	s.GuardianPower.Encode(w)
	w.Uint32(s.InventoryVersion)
	if s.InventoryVersion == 109 {
		w.Uint16(uint16(len(s.Inventory)))
		for _, i := range s.Inventory {
			i.encodeCompact(w)
		}
	} else {
		writeList(w, s.Inventory)
	}
}

func (s PlayerState) Validate() error {
	if s.PlayerVersion != supportedPlayerVersion && s.PlayerVersion != 33 {
		return fmt.Errorf("unsupported player version %d", s.PlayerVersion)
	}
	if s.InventoryVersion != supportedInventoryVersion && s.InventoryVersion != 109 {
		return fmt.Errorf("unsupported inventory version %d", s.InventoryVersion)
	}
	if s.InventoryVersion == 109 {
		if len(s.Inventory) > 65535 {
			return fmt.Errorf("inventory exceeds compact count")
		}
		for _, i := range s.Inventory {
			if err := i.validateCompact(); err != nil {
				return err
			}
		}
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
	KnownBiomeNames  []string    `json:"knownBiomeNames,omitempty"`
	BuildMenu        []byte      `json:"buildMenu,omitempty"`
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

func (t *PlayerTail) Decode(r *binary.Reader) {
	t.decodeVersion(r, 29)
}

func (t *PlayerTail) decodeVersion(r *binary.Reader, version uint32) {
	t.KnownRecipes = readStringList(r)
	t.KnownStations = readList[Station](r)
	t.KnownMaterials = readStringList(r)
	t.ShownTutorials = readStringList(r)
	t.Uniques = readStringList(r)
	t.Trophies = readStringList(r)
	if version == 33 {
		t.KnownBiomeNames = readStringList(r)
	} else {
		t.KnownBiomes = readList[Biome](r)
	}
	t.PlayerKnownTexts = readList[TextEntry](r)

	t.Beard = r.String()
	t.Hair = r.String()
	t.SkinColor.Decode(r)
	t.HairColor.Decode(r)
	t.ModelIndex = r.Uint32()

	t.Foods = readList[Food](r)

	t.SkillVersion = r.Uint32()
	if t.SkillVersion != 2 {
		panic(fmt.Errorf("unsupported skill version %d", t.SkillVersion))
	}
	t.Skills = readList[Skill](r)
	t.CustomData = readList[TextEntry](r)

	if version == 33 && r.Remaining() < 16 {
		panic("fch: truncated version 33 player tail")
	}
	if r.Remaining() >= 8 {
		t.Stamina = r.Float32()
		t.MaxEitr = r.Float32()
		t.tailFloatCount = 2
	}
	if r.Remaining() >= 4 {
		t.Eitr = r.Float32()
		t.tailFloatCount = 3
	}
	if version == 33 {
		t.BuildMenu = append([]byte(nil), r.Bytes(int(r.Uint32()))...)
	}
}

func (t PlayerTail) Encode(w *binary.Writer) {
	t.encodeVersion(w, 29)
}

func (t PlayerTail) encodeVersion(w *binary.Writer, version uint32) {
	writeStringList(w, t.KnownRecipes)
	writeList(w, t.KnownStations)
	writeStringList(w, t.KnownMaterials)
	writeStringList(w, t.ShownTutorials)
	writeStringList(w, t.Uniques)
	writeStringList(w, t.Trophies)
	if version == 33 {
		writeStringList(w, t.KnownBiomeNames)
	} else {
		writeList(w, t.KnownBiomes)
	}
	writeList(w, t.PlayerKnownTexts)

	w.String(t.Beard)
	w.String(t.Hair)
	t.SkinColor.Encode(w)
	t.HairColor.Encode(w)
	w.Uint32(t.ModelIndex)

	writeList(w, t.Foods)

	w.Uint32(t.SkillVersion)
	writeList(w, t.Skills)
	writeList(w, t.CustomData)

	tailFloatCount := t.normalizedTailFloatCount()
	if tailFloatCount >= 2 {
		w.Float32(t.Stamina)
		w.Float32(t.MaxEitr)
	}
	if tailFloatCount >= 3 {
		w.Float32(t.Eitr)
	}
	if version == 33 {
		w.Uint32(uint32(len(t.BuildMenu)))
		w.Bytes(t.BuildMenu)
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
