package fch

import (
	"math"
	"strings"
)

func readList[T any](r *reader, read func(*reader) T) []T {
	count := r.u32()
	out := make([]T, 0, count)
	for range count {
		out = append(out, read(r))
	}
	return out
}

func statEntry(r *reader) StatEntry {
	return StatEntry{Name: r.str(), Value: r.f32()}
}

func timedEntry(r *reader) TimedEntry {
	return TimedEntry{Name: r.str(), Seconds: r.f32()}
}

func textEntry(r *reader) TextEntry {
	return TextEntry{Key: r.str(), Value: r.str()}
}

func station(r *reader) Station {
	return Station{Name: r.str(), Level: r.u32()}
}

func biome(r *reader) uint32 {
	return r.u32()
}

func food(r *reader) Food {
	return Food{Name: r.str(), Time: r.f32()}
}

func skill(r *reader) Skill {
	skillType := r.i32()
	level := r.f32()
	return Skill{
		Type:         skillType,
		Name:         skillName(skillType),
		Level:        level,
		DisplayLevel: int32(math.Floor(float64(level))),
		Accumulator:  r.f32(),
	}
}

func worldKey(r *reader) WorldKey {
	raw := r.str()
	seconds := r.f32()
	key, setting, ok := strings.Cut(raw, " ")
	if !ok {
		return WorldKey{Raw: raw, Seconds: seconds}
	}
	return WorldKey{Raw: raw, Key: key, Setting: setting, Seconds: seconds}
}
