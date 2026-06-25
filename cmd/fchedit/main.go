package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	fch "github.com/lanchelms/fch-decoder"
)

type editOp interface {
	apply(*fch.Character) error
}

type opFlag struct {
	ops   *[]editOp
	newOp func(string) (editOp, error)
}

func (f opFlag) String() string {
	return ""
}

func (f opFlag) Set(value string) error {
	op, err := f.newOp(value)
	if err != nil {
		return err
	}
	*f.ops = append(*f.ops, op)
	return nil
}

type inventoryAction int

const (
	addInventory inventoryAction = iota
	removeInventory
)

type inventoryOp struct {
	action inventoryAction
	item   fch.Item
}

func (op inventoryOp) apply(c *fch.Character) error {
	switch op.action {
	case addInventory:
		c.AddInventoryItem(op.item)
		return nil
	case removeInventory:
		return c.RemoveInventoryItem(op.item.Name)
	default:
		return fmt.Errorf("unknown inventory action %d", op.action)
	}
}

func newInventoryOp(action inventoryAction) func(string) (editOp, error) {
	return func(value string) (editOp, error) {
		item, err := parseInventoryAction(action, value)
		if err != nil {
			return nil, err
		}
		return inventoryOp{action: action, item: item}, nil
	}
}

type setSkillLevelOp struct {
	skillType int32
	name      string
	level     float32
}

func (op setSkillLevelOp) apply(c *fch.Character) error {
	c.SetSkillLevel(op.skillType, op.name, op.level)
	return nil
}

type setPlayerStatOp struct {
	index int
	name  string
	value float32
}

func (op setPlayerStatOp) apply(c *fch.Character) error {
	return c.SetPlayerStat(op.index, op.name, op.value)
}

type statSetter func(*fch.Character, string, float32)

type setStatOp struct {
	assignment assignment
	set        statSetter
}

func (op setStatOp) apply(c *fch.Character) error {
	op.set(c, op.assignment.name, op.assignment.value)
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
	fs.Var(opFlag{ops: &ops, newOp: newInventoryOp(addInventory)}, "add-inventory", "add inventory item: name[,stack=n,durability=n,grid-x=n,grid-y=n,equipped=bool,quality=n,variant=n,crafter-id=n,crafter-name=s,world-level=n,picked-up=bool]")
	fs.Var(opFlag{ops: &ops, newOp: newInventoryOp(removeInventory)}, "remove-inventory", "remove first inventory item with exact name")
	fs.Var(opFlag{ops: &ops, newOp: newSetSkillLevelOp}, "set-skill-level", "set skill level: skill-name-or-type=level")
	fs.Var(opFlag{ops: &ops, newOp: newSetStatOp((*fch.Character).UpsertEnemyStat)}, "set-enemy-stat", "set enemy stat: name=value")
	fs.Var(opFlag{ops: &ops, newOp: newSetStatOp((*fch.Character).UpsertMaterialStat)}, "set-material-stat", "set material stat: name=value")
	fs.Var(opFlag{ops: &ops, newOp: newSetPlayerStatOp}, "set-player-stat", "set player stat: stat-name-or-index=value")
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

func newSetSkillLevelOp(value string) (editOp, error) {
	assignment, err := parseAssignment(value)
	if err != nil {
		return nil, err
	}
	skill, err := parseSkillRef(assignment.name)
	if err != nil {
		return nil, err
	}
	return setSkillLevelOp{skillType: skill.skillType, name: skill.name, level: assignment.value}, nil
}

func newSetPlayerStatOp(value string) (editOp, error) {
	assignment, err := parseAssignment(value)
	if err != nil {
		return nil, err
	}
	stat, err := parsePlayerStatRef(assignment.name)
	if err != nil {
		return nil, err
	}
	return setPlayerStatOp{index: stat.index, name: stat.name, value: assignment.value}, nil
}

func newSetStatOp(set statSetter) func(string) (editOp, error) {
	return func(value string) (editOp, error) {
		assignment, err := parseAssignment(value)
		if err != nil {
			return nil, err
		}
		return setStatOp{assignment: assignment, set: set}, nil
	}
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
