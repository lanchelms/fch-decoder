package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	fch "github.com/brian/fch-decoder"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const defaultCacheTTL = 5 * time.Second

var (
	skillsDesc = prometheus.NewDesc(
		"valheim_character_skills",
		"Valheim character skill levels.",
		[]string{"player", "skill"},
		nil,
	)
	craftingDesc = prometheus.NewDesc(
		"valheim_character_crafting",
		"Valheim character recipe stat counters.",
		[]string{"player", "recipe"},
		nil,
	)
	enemiesDesc = prometheus.NewDesc(
		"valheim_character_enemies",
		"Valheim character enemy stat counters.",
		[]string{"player", "enemy"},
		nil,
	)
	statsDesc = prometheus.NewDesc(
		"valheim_character_stats",
		"Selected Valheim character player stat counters.",
		[]string{"player", "stat"},
		nil,
	)
	distanceDesc = prometheus.NewDesc(
		"valheim_character_distance",
		"Valheim character distance traveled in world units. Sail is inferred because Valheim's raw DistanceSail stat is a sample counter.",
		[]string{"player", "mode"},
		nil,
	)
	scrapeErrorsDesc = prometheus.NewDesc(
		"valheim_character_scrape_errors",
		"Number of Valheim character files or directories that could not be scraped.",
		nil,
		nil,
	)
	allDescs = []*prometheus.Desc{
		skillsDesc,
		craftingDesc,
		enemiesDesc,
		statsDesc,
		distanceDesc,
		scrapeErrorsDesc,
	}
)

var allowedPlayerStats = map[string]bool{
	"Deaths":                true,
	"CraftsOrUpgrades":      true,
	"Builds":                true,
	"Jumps":                 true,
	"EnemyHits":             true,
	"EnemyKills":            true,
	"EnemyKillsLastHits":    true,
	"HitsTakenEnemies":      true,
	"ItemsPickedUp":         true,
	"Crafts":                true,
	"Upgrades":              true,
	"TimeInBase":            true,
	"TimeOutOfBase":         true,
	"Sleep":                 true,
	"ItemStandUses":         true,
	"ArmorStandUses":        true,
	"WorldLoads":            true,
	"TreeChops":             true,
	"Tree":                  true,
	"LogChops":              true,
	"Logs":                  true,
	"MineHits":              true,
	"Mines":                 true,
	"RavenTalk":             true,
	"RavenAppear":           true,
	"CreatureTamed":         true,
	"ArrowsShot":            true,
	"TombstonesOpenedOwn":   true,
	"TombstonesOpenedOther": true,
	"TombstonesFit":         true,
	"DoorsOpened":           true,
	"DoorsClosed":           true,
	"BeesHarvested":         true,
	"SapHarvested":          true,
	"TurretAmmoAdded":       true,
	"TurretTrophySet":       true,
	"TrapArmed":             true,
	"TrapTriggered":         true,
	"PlaceStacks":           true,
	"PortalDungeonIn":       true,
	"PortalDungeonOut":      true,
	"BossKills":             true,
	"BossLastHits":          true,
}

type collector struct {
	dir      string
	workers  int
	cacheTTL time.Duration

	mu       sync.Mutex
	cachedAt time.Time
	cached   snapshot
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range allDescs {
		ch <- desc
	}
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	snap := c.getSnapshot()
	for _, character := range snap.characters {
		for _, sample := range character.samples {
			ch <- prometheus.MustNewConstMetric(sample.desc, prometheus.GaugeValue, sample.value, sample.labels...)
		}
	}
	ch <- prometheus.MustNewConstMetric(scrapeErrorsDesc, prometheus.GaugeValue, float64(snap.errors))
}

func (c *collector) getSnapshot() snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if c.cacheTTL > 0 && !c.cachedAt.IsZero() && now.Sub(c.cachedAt) < c.cacheTTL {
		return c.cached
	}

	snap := loadSnapshot(c.dir, c.workers)
	c.cached = snap
	c.cachedAt = now
	return snap
}

type snapshot struct {
	characters []metrics
	errors     int
}

type metrics struct {
	player  string
	samples []sample
}

func (m *metrics) addStats(desc *prometheus.Desc, entries []fch.StatEntry) {
	for _, entry := range entries {
		name := cleanMetricLabel(entry.Name, desc)
		if name == "" {
			continue
		}
		m.add(desc, float64(entry.Value), name)
	}
}

func (m *metrics) add(desc *prometheus.Desc, value float64, label string) {
	m.samples = append(m.samples, sample{
		desc:   desc,
		value:  value,
		labels: []string{m.player, label},
	})
}

type sample struct {
	desc   *prometheus.Desc
	value  float64
	labels []string
}

func main() {
	addr := flag.String("addr", ":9108", "address to serve Prometheus metrics on")
	dir := flag.String("dir", "", "Valheim characters_local directory")
	metricsPath := flag.String("metrics-path", "/metrics", "Prometheus metrics path")
	workers := flag.Int("workers", runtime.NumCPU(), "maximum number of character files to decode in parallel")
	cacheTTL := flag.Duration("cache-ttl", defaultCacheTTL, "how long to reuse decoded metrics between scrapes")
	flag.Parse()
	if *dir == "" {
		log.Fatal("missing required -dir flag")
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(&collector{
		dir:      *dir,
		workers:  *workers,
		cacheTTL: *cacheTTL,
	})

	mux := http.NewServeMux()
	mux.Handle(*metricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Valheim character metrics at %s\n", *metricsPath)
	})

	log.Printf("serving Valheim character metrics on %s from %s", *addr, *dir)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

func loadSnapshot(dir string, workers int) snapshot {
	paths, err := characterFiles(dir)
	if err != nil {
		log.Printf("cannot read character directory %s: %v", dir, err)
		return snapshot{errors: 1}
	}
	if workers < 1 {
		workers = 1
	}
	if workers > len(paths) && len(paths) > 0 {
		workers = len(paths)
	}

	pathCh := make(chan string)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var snap snapshot

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range pathCh {
				metrics, err := loadMetrics(path)
				mu.Lock()
				if err != nil {
					log.Printf("cannot scrape character file %s: %v", path, err)
					snap.errors++
				} else {
					snap.characters = append(snap.characters, metrics)
				}
				mu.Unlock()
			}
		}()
	}

	for _, path := range paths {
		pathCh <- path
	}
	close(pathCh)

	wg.Wait()
	return snap
}

func loadMetrics(path string) (metrics, error) {
	f, err := os.Open(path)
	if err != nil {
		return metrics{}, err
	}
	defer f.Close()

	character, err := fch.Decode(f)
	if err != nil {
		return metrics{}, err
	}
	return newMetrics(character), nil
}

func newMetrics(character *fch.Character) metrics {
	out := metrics{player: character.Player.Name}
	for _, skill := range character.Player.Skills {
		if skill.Name == "" {
			continue
		}
		out.add(skillsDesc, float64(skill.DisplayLevel), skill.Name)
	}
	out.addStats(craftingDesc, character.Player.RecipeStats)
	out.addStats(enemiesDesc, character.Player.EnemyStats)
	for _, stat := range character.PlayerStats {
		if !allowedPlayerStats[stat.Name] {
			continue
		}
		out.add(statsDesc, float64(stat.Value), stat.Name)
	}
	out.addDistanceStats(character.PlayerStats)
	return out
}

func (m *metrics) addDistanceStats(entries []fch.StatEntry) {
	stats := map[string]float64{}
	for _, entry := range entries {
		stats[entry.Name] = float64(entry.Value)
	}

	total, ok := stats["DistanceTraveled"]
	if !ok {
		return
	}
	walk := stats["DistanceWalk"]
	run := stats["DistanceRun"]
	air := stats["DistanceAir"]
	sail := total - walk - run - air
	if sail < 0 && sail > -0.001 {
		sail = 0
	}

	m.add(distanceDesc, total, "Total")
	m.add(distanceDesc, walk, "Walk")
	m.add(distanceDesc, run, "Run")
	m.add(distanceDesc, sail, "Sail")
	m.add(distanceDesc, air, "Air")
}

func characterFiles(dir string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.fch"))
	if err != nil {
		return nil, err
	}

	current := paths[:0]
	for _, path := range paths {
		if strings.Contains(filepath.Base(path), "backup_auto-") {
			continue
		}
		current = append(current, path)
	}
	sort.Strings(current)
	return current, nil
}

func cleanMetricLabel(value string, desc *prometheus.Desc) string {
	value = strings.ReplaceAll(value, "$", "")
	switch desc {
	case craftingDesc:
		return titleName(strings.TrimPrefix(value, "item_"))
	case enemiesDesc:
		return titleName(strings.TrimPrefix(value, "enemy_"))
	default:
		return value
	}
}

func titleName(value string) string {
	title := cases.Title(language.Und)
	parts := strings.Split(value, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = title.String(part)
	}
	return strings.Join(parts, "")
}
