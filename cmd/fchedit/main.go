package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	fch "github.com/lanchelms/fch-decoder"
)

type editOp interface {
	apply(*fch.Character) error
}

type opFlag struct {
	ops   *[]editOp
	parse func(string) (editOp, error)
}

func (f opFlag) String() string {
	return ""
}

func (f opFlag) Set(value string) error {
	op, err := f.parse(value)
	if err != nil {
		return err
	}
	*f.ops = append(*f.ops, op)
	return nil
}

type addInventoryOp struct {
	item fch.Item
}

func (op addInventoryOp) apply(c *fch.Character) error {
	c.Player.Inventory = append(c.Player.Inventory, op.item)
	return nil
}

type removeInventoryOp struct {
	name string
}

func (op removeInventoryOp) apply(c *fch.Character) error {
	for i, item := range c.Player.Inventory {
		if item.Name == op.name {
			c.Player.Inventory = append(c.Player.Inventory[:i], c.Player.Inventory[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("inventory item %q not found", op.name)
}

type setSkillLevelOp struct {
	skillType int32
	name      string
	level     float32
}

func (op setSkillLevelOp) apply(c *fch.Character) error {
	for i := range c.Player.Skills {
		if c.Player.Skills[i].Type == op.skillType {
			c.Player.Skills[i].Level = op.level
			return nil
		}
	}
	c.Player.Skills = append(c.Player.Skills, fch.Skill{
		Type:  op.skillType,
		Name:  op.name,
		Level: op.level,
	})
	return nil
}

type setPlayerStatOp struct {
	index int
	name  string
	value float32
}

func (op setPlayerStatOp) apply(c *fch.Character) error {
	for len(c.PlayerStats) <= op.index {
		c.PlayerStats = append(c.PlayerStats, fch.StatEntry{})
	}
	c.PlayerStats[op.index] = fch.StatEntry{Name: op.name, Value: op.value}
	c.PlayerStatCount = uint32(len(c.PlayerStats))
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	var ops []editOp
	var outPath string
	var inPlace bool

	fs := flag.NewFlagSet("fchedit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&outPath, "out", "", "path to write the edited character file")
	fs.BoolVar(&inPlace, "in-place", false, "overwrite the input character file")
	fs.Var(opFlag{ops: &ops, parse: parseAddInventory}, "add-inventory", "add inventory item: name[,stack=n,durability=n,grid-x=n,grid-y=n,equipped=bool,quality=n,variant=n,crafter-id=n,crafter-name=s,world-level=n,picked-up=bool]")
	fs.Var(opFlag{ops: &ops, parse: parseRemoveInventory}, "remove-inventory", "remove first inventory item with exact name")
	fs.Var(opFlag{ops: &ops, parse: parseSetSkillLevel}, "set-skill-level", "set skill level: skill-name-or-type=level")
	fs.Var(opFlag{ops: &ops, parse: parseSetEnemyStat}, "set-enemy-stat", "set enemy stat: name=value")
	fs.Var(opFlag{ops: &ops, parse: parseSetMaterialStat}, "set-material-stat", "set material stat: name=value")
	fs.Var(opFlag{ops: &ops, parse: parseSetPlayerStat}, "set-player-stat", "set player stat: stat-name-or-index=value")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: fchedit [flags] <character.fch>")
	}
	if len(ops) == 0 {
		return fmt.Errorf("no edits requested")
	}
	if inPlace && outPath != "" {
		return fmt.Errorf("-in-place and -out cannot be used together")
	}
	if !inPlace && outPath == "" {
		return fmt.Errorf("missing required -out flag; use -in-place to overwrite the input file")
	}

	inputPath := fs.Arg(0)
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	character, err := fch.DecodeBytes(data)
	if err != nil {
		return err
	}
	for _, op := range ops {
		if err := op.apply(character); err != nil {
			return err
		}
	}
	encoded, err := fch.EncodeBytes(character)
	if err != nil {
		return err
	}

	target := outPath
	if inPlace {
		target = inputPath
	}
	if err := writeFile(target, encoded, inputPath); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "wrote %s\n", target)
	return nil
}

func parseAddInventory(value string) (editOp, error) {
	parts := strings.Split(value, ",")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return nil, fmt.Errorf("add inventory item name is required")
	}
	item := fch.Item{
		Name:       name,
		Stack:      1,
		Durability: 1,
		Quality:    1,
		PickedUp:   true,
	}

	for _, part := range parts[1:] {
		key, raw, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("invalid add inventory field %q", part)
		}
		key = strings.TrimSpace(key)
		raw = strings.TrimSpace(raw)
		var err error
		switch key {
		case "stack":
			item.Stack, err = parseI32(raw)
		case "durability":
			item.Durability, err = parseF32(raw)
		case "grid-x":
			item.GridX, err = parseI32(raw)
		case "grid-y":
			item.GridY, err = parseI32(raw)
		case "equipped":
			item.Equipped, err = strconv.ParseBool(raw)
		case "quality":
			item.Quality, err = parseI32(raw)
		case "variant":
			item.Variant, err = parseI32(raw)
		case "crafter-id":
			item.CrafterID, err = strconv.ParseUint(raw, 10, 64)
		case "crafter-name":
			item.CrafterName = raw
		case "world-level":
			item.WorldLevel, err = parseU32(raw)
		case "picked-up":
			item.PickedUp, err = strconv.ParseBool(raw)
		default:
			return nil, fmt.Errorf("unknown add inventory field %q", key)
		}
		if err != nil {
			return nil, fmt.Errorf("invalid add inventory field %q: %w", key, err)
		}
	}
	return addInventoryOp{item: item}, nil
}

func parseRemoveInventory(value string) (editOp, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return nil, fmt.Errorf("remove inventory item name is required")
	}
	return removeInventoryOp{name: name}, nil
}

func parseSetSkillLevel(value string) (editOp, error) {
	name, level, err := parseAssignment(value)
	if err != nil {
		return nil, err
	}
	skillType, skillName, err := parseSkillType(name)
	if err != nil {
		return nil, err
	}
	return setSkillLevelOp{skillType: skillType, name: skillName, level: level}, nil
}

func parseSetEnemyStat(value string) (editOp, error) {
	name, amount, err := parseAssignment(value)
	if err != nil {
		return nil, err
	}
	return namedStatOp{
		selectEntries: func(c *fch.Character) *[]fch.StatEntry { return &c.Player.EnemyStats },
		name:          name,
		value:         amount,
	}, nil
}

func parseSetMaterialStat(value string) (editOp, error) {
	name, amount, err := parseAssignment(value)
	if err != nil {
		return nil, err
	}
	return namedStatOp{
		selectEntries: func(c *fch.Character) *[]fch.StatEntry { return &c.Player.MaterialStats },
		name:          name,
		value:         amount,
	}, nil
}

func parseSetPlayerStat(value string) (editOp, error) {
	name, amount, err := parseAssignment(value)
	if err != nil {
		return nil, err
	}
	index, statName, err := parsePlayerStatIndex(name)
	if err != nil {
		return nil, err
	}
	return setPlayerStatOp{index: index, name: statName, value: amount}, nil
}

type namedStatOp struct {
	selectEntries func(*fch.Character) *[]fch.StatEntry
	name          string
	value         float32
}

func (op namedStatOp) apply(c *fch.Character) error {
	upsertStat(op.selectEntries(c), op.name, op.value)
	return nil
}

func parseAssignment(value string) (string, float32, error) {
	name, raw, ok := strings.Cut(value, "=")
	if !ok {
		return "", 0, fmt.Errorf("expected name=value, got %q", value)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", 0, fmt.Errorf("assignment name is required")
	}
	amount, err := parseF32(strings.TrimSpace(raw))
	if err != nil {
		return "", 0, err
	}
	return name, amount, nil
}

func parseSkillType(value string) (int32, string, error) {
	if n, err := strconv.ParseInt(value, 10, 32); err == nil {
		return int32(n), value, nil
	}
	skillType, ok := fch.SkillTypeByName(value)
	if !ok {
		return 0, "", fmt.Errorf("unknown skill %q", value)
	}
	return skillType, value, nil
}

func parsePlayerStatIndex(value string) (int, string, error) {
	if n, err := strconv.ParseInt(value, 10, 32); err == nil {
		if n < 0 {
			return 0, "", fmt.Errorf("invalid player stat index %d", n)
		}
		return int(n), value, nil
	}
	index, ok := fch.PlayerStatIndexByName(value)
	if !ok {
		return 0, "", fmt.Errorf("unknown player stat %q", value)
	}
	return index, value, nil
}

func upsertStat(entries *[]fch.StatEntry, name string, value float32) {
	for i := range *entries {
		if strings.EqualFold((*entries)[i].Name, name) {
			(*entries)[i].Value = value
			return
		}
	}
	*entries = append(*entries, fch.StatEntry{Name: name, Value: value})
}

func parseF32(value string) (float32, error) {
	f, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return 0, err
	}
	return float32(f), nil
}

func parseI32(value string) (int32, error) {
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(n), nil
}

func parseU32(value string) (uint32, error) {
	n, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(n), nil
}

func writeFile(path string, data []byte, modeFrom string) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(modeFrom); err == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
