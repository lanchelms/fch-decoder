package valheim

import (
	"fmt"
	"github.com/lanchelms/fch-decoder/binary"
)

var statGroupNames = []string{"RawStats", "Any", "Hammer", "Casual", "VeryEasy", "Easy", "Default", "Hard", "VeryHard", "Hardcore"}

// StatGroup retains one saved statistics group. Groups overlap and must not be summed.
type StatGroup struct {
	Name           string        `json:"name"`
	Stats          []StatEntry   `json:"stats"`
	KnownWorlds    []TimeEntry   `json:"knownWorlds,omitempty"`
	KnownWorldKeys []WorldKey    `json:"knownWorldKeys,omitempty"`
	KnownCommands  []StatEntry   `json:"-"`
	EnemyStats     [][]StatEntry `json:"enemyStats"`
	ItemsPickedUp  []StatEntry   `json:"itemsPickedUp,omitempty"`
	ItemsCrafted   []StatEntry   `json:"itemsCrafted,omitempty"`
	Pickables      []StatEntry   `json:"pickables,omitempty"`
	FoodsEaten     []StatEntry   `json:"foodsEaten,omitempty"`
	PiecesPlaced   []StatEntry   `json:"piecesPlaced,omitempty"`
}

func (g *StatGroup) Decode(r *binary.Reader) {
	defer func() {
		if failure := recover(); failure != nil {
			panic(fmt.Errorf("stat group %s: %v", g.Name, failure))
		}
	}()
	for _, name := range currentPlayerStatNames {
		g.Stats = append(g.Stats, StatEntry{Name: name, Value: r.Float32()})
	}
	g.KnownWorlds = readList[TimeEntry](r)
	g.KnownWorldKeys = readList[WorldKey](r)
	g.KnownCommands = readList[StatEntry](r)
	if n := r.Uint32(); n != 5 {
		panic(fmt.Errorf("expected five enemy categories, got %d", n))
	}
	for range 5 {
		g.EnemyStats = append(g.EnemyStats, readList[StatEntry](r))
	}
	g.ItemsPickedUp = readList[StatEntry](r)
	g.ItemsCrafted = readList[StatEntry](r)
	g.Pickables = readList[StatEntry](r)
	g.FoodsEaten = readList[StatEntry](r)
	g.PiecesPlaced = readList[StatEntry](r)
}

func (g StatGroup) Encode(w *binary.Writer) {
	for _, s := range g.Stats {
		w.Float32(s.Value)
	}
	writeList(w, g.KnownWorlds)
	writeList(w, g.KnownWorldKeys)
	writeList(w, g.KnownCommands)
	w.Uint32(uint32(len(g.EnemyStats)))
	for _, entries := range g.EnemyStats {
		writeList(w, entries)
	}
	writeList(w, g.ItemsPickedUp)
	writeList(w, g.ItemsCrafted)
	writeList(w, g.Pickables)
	writeList(w, g.FoodsEaten)
	writeList(w, g.PiecesPlaced)
}
