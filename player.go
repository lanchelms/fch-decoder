package fch

import "time"

type PlayerData struct {
	Name             string        `json:"name"`
	PlayerID         uint64        `json:"playerId"`
	StartSeed        string        `json:"startSeed"`
	UsedCheats       bool          `json:"usedCheats"`
	DateCreatedUnix  int64         `json:"dateCreatedUnix"`
	KnownWorlds      []TimedEntry  `json:"knownWorlds,omitempty"`
	KnownWorldKeys   []WorldKey    `json:"knownWorldKeys,omitempty"`
	KnownCommands    []StatEntry   `json:"-"`
	EnemyStats       []StatEntry   `json:"enemyStats,omitempty"`
	MaterialStats    []StatEntry   `json:"materialStats,omitempty"`
	RecipeStats      []StatEntry   `json:"recipeStats,omitempty"`
	GuardianPower    GuardianPower `json:"guardianPower"`
	HasPlayerData    bool          `json:"hasPlayerData"`
	PlayerDataLength uint32        `json:"playerDataLength"`
	PlayerVersion    uint32        `json:"playerVersion"`
	MaxHealth        float32       `json:"maxHealth"`
	Health           float32       `json:"health"`
	MaxStamina       float32       `json:"maxStamina"`
	Stamina          float32       `json:"stamina"`
	MaxEitr          float32       `json:"maxEitr"`
	Eitr             float32       `json:"eitr"`
	TimeSinceDeath   float32       `json:"timeSinceDeath"`
	InventoryVersion uint32        `json:"inventoryVersion"`
	Inventory        []Item        `json:"inventory,omitempty"`
	KnownRecipes     []string      `json:"knownRecipes,omitempty"`
	KnownStations    []Station     `json:"knownStations,omitempty"`
	KnownMaterials   []string      `json:"knownMaterials,omitempty"`
	ShownTutorials   []string      `json:"-"`
	Uniques          []string      `json:"uniques,omitempty"`
	Trophies         []string      `json:"trophies,omitempty"`
	KnownBiomes      []uint32      `json:"knownBiomes,omitempty"`
	PlayerKnownTexts []TextEntry   `json:"-"`
	Beard            string        `json:"beard,omitempty"`
	Hair             string        `json:"hair,omitempty"`
	SkinColor        Vector3       `json:"skinColor"`
	HairColor        Vector3       `json:"hairColor"`
	ModelIndex       uint32        `json:"modelIndex"`
	Foods            []Food        `json:"foods,omitempty"`
	SkillVersion     uint32        `json:"skillVersion,omitempty"`
	Skills           []Skill       `json:"skills,omitempty"`
	CustomData       []TextEntry   `json:"customData,omitempty"`
	tailFloatCount   int
}

func NewPlayer(name string, playerID uint64) PlayerData {
	return PlayerData{
		Name:             name,
		PlayerID:         playerID,
		DateCreatedUnix:  time.Now().Unix(),
		HasPlayerData:    true,
		PlayerVersion:    supportedPlayerVersion,
		InventoryVersion: supportedInventoryVersion,
		SkillVersion:     supportedSkillVersion,
	}
}

func NewPlayerStats() []StatEntry {
	stats := make([]StatEntry, len(playerStatNames))
	for i, name := range playerStatNames {
		stats[i].Name = name
	}
	return stats
}
