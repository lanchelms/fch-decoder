package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	fch "github.com/lanchelms/fch-decoder"
)

const characterEnv = "CHARACTER"

type cli struct {
	Character string    `name:"character" env:"CHARACTER" required:"" type:"path" help:"Character file to edit."`
	Out       string    `name:"out" type:"path" help:"Write the edited character to this path instead of updating the input file."`
	Add       addCmd    `cmd:"" help:"Add character data."`
	Remove    removeCmd `cmd:"" help:"Remove character data."`
	Set       setCmd    `cmd:"" help:"Set character data."`
}

type addCmd struct {
	Inventory addInventoryCmd `cmd:"" help:"Add an inventory item."`
}

type addInventoryCmd struct {
	Item string `arg:"" name:"item" help:"Item spec: name[,stack=n,durability=n,grid-x=n,grid-y=n,equipped=bool,quality=n,variant=n,crafter-id=n,crafter-name=s,world-level=n,picked-up=bool]."`
}

func (cmd *addInventoryCmd) Run(r *editRunner) error {
	item, err := parseInventoryItem(cmd.Item)
	if err != nil {
		return err
	}
	return r.apply(func(c *fch.Character) error {
		c.AddInventoryItem(item)
		return nil
	})
}

type removeCmd struct {
	Inventory removeInventoryCmd `cmd:"" help:"Remove an inventory item."`
}

type removeInventoryCmd struct {
	Name string `arg:"" name:"name" help:"Inventory item name to remove."`
}

func (cmd *removeInventoryCmd) Run(r *editRunner) error {
	name, err := parseInventoryName(cmd.Name)
	if err != nil {
		return err
	}
	return r.apply(func(c *fch.Character) error {
		return c.RemoveInventoryItem(name)
	})
}

type setCmd struct {
	Skill      setSkillCmd      `cmd:"" help:"Set a skill level."`
	Enemy      setEnemyCmd      `cmd:"" help:"Set an enemy stat."`
	Material   setMaterialCmd   `cmd:"" help:"Set a material stat."`
	PlayerStat setPlayerStatCmd `cmd:"" name:"player-stat" help:"Set a player stat."`
}

type setSkillCmd struct {
	Skill string  `arg:"" name:"skill" help:"Skill name or numeric type."`
	Level float32 `arg:"" name:"level" help:"Skill level."`
}

func (cmd *setSkillCmd) Run(r *editRunner) error {
	skill, err := parseSkillRef(cmd.Skill)
	if err != nil {
		return err
	}
	return r.apply(func(c *fch.Character) error {
		c.SetSkillLevel(skill.skillType, skill.name, cmd.Level)
		return nil
	})
}

type setEnemyCmd struct {
	Name  string  `arg:"" name:"name" help:"Enemy stat name."`
	Value float32 `arg:"" name:"value" help:"Enemy stat value."`
}

func (cmd *setEnemyCmd) Run(r *editRunner) error {
	return r.apply(func(c *fch.Character) error {
		c.UpsertEnemyStat(cmd.Name, cmd.Value)
		return nil
	})
}

type setMaterialCmd struct {
	Name  string  `arg:"" name:"name" help:"Material stat name."`
	Value float32 `arg:"" name:"value" help:"Material stat value."`
}

func (cmd *setMaterialCmd) Run(r *editRunner) error {
	return r.apply(func(c *fch.Character) error {
		c.UpsertMaterialStat(cmd.Name, cmd.Value)
		return nil
	})
}

type setPlayerStatCmd struct {
	Stat  string  `arg:"" name:"stat" help:"Player stat name or numeric index."`
	Value float32 `arg:"" name:"value" help:"Player stat value."`
}

func (cmd *setPlayerStatCmd) Run(r *editRunner) error {
	stat, err := parsePlayerStatRef(cmd.Stat)
	if err != nil {
		return err
	}
	return r.apply(func(c *fch.Character) error {
		return c.SetPlayerStat(stat.index, stat.name, cmd.Value)
	})
}

type editRunner struct {
	path   string
	out    string
	stdout io.Writer
}

func (r *editRunner) apply(edit func(*fch.Character) error) error {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}
	character, err := fch.DecodeBytes(data)
	if err != nil {
		return err
	}
	if err := edit(character); err != nil {
		return err
	}
	encoded, err := fch.EncodeBytes(character)
	if err != nil {
		return err
	}

	target := r.out
	if target == "" {
		target = r.path
	}
	if err := writeFile(target, encoded, r.path); err != nil {
		return err
	}
	fmt.Fprintf(r.stdout, "wrote %s\n", target)
	return nil
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

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	var cli cli
	parser, err := kong.New(
		&cli,
		kong.Name("fchedit"),
		kong.Description("Edit a Valheim character file."),
		kong.Writers(stdout, stderr),
	)
	if err != nil {
		return err
	}
	ctx, err := parser.Parse(args)
	if err != nil {
		return err
	}
	runner := &editRunner{path: cli.Character, out: cli.Out, stdout: stdout}
	return ctx.Run(runner)
}
