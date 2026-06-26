package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/alecthomas/kong"
	fch "github.com/lanchelms/fch-decoder"
)

const characterEnv = "CHARACTER"

type cli struct {
	Character string    `name:"character" env:"CHARACTER" type:"path" help:"Character file to edit."`
	Out       string    `name:"out" type:"path" help:"Write the edited character to this path instead of updating the input file."`
	DryRun    bool      `name:"dry-run" help:"Decode, validate, and summarize the edit without writing a file."`
	NoBackup  bool      `name:"no-backup" help:"Do not create a backup before editing a character in place."`
	Add       addCmd    `cmd:"" help:"Add character data."`
	Remove    removeCmd `cmd:"" help:"Remove character data."`
	Set       setCmd    `cmd:"" help:"Set character data."`
	List      listCmd   `cmd:"" help:"List editable names or character data."`
}

type addCmd struct {
	Inventory addInventoryCmd `cmd:"" help:"Add an inventory item."`
}

type addInventoryCmd struct {
	Item string `arg:"" name:"item" help:"Item spec: name[,stack=n,durability=n,pos=x:y,equipped=bool,quality=n,variant=n,crafter-id=n,crafter-name=s,world-level=n,picked-up=bool]."`
}

func (cmd *addInventoryCmd) Run(r *editRunner) error {
	item, positioned, err := parseInventoryItem(cmd.Item)
	if err != nil {
		return err
	}
	return r.apply(func(c *fch.Character) error {
		if positioned {
			return c.PutInventoryItem(item, true)
		}
		return c.PlaceInventoryItem(item)
	}, fmt.Sprintf("add inventory %s", item.Name))
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
	}, fmt.Sprintf("remove inventory %s", name))
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
	level, err := parseSkillLevel(cmd.Level)
	if err != nil {
		return err
	}
	return r.apply(func(c *fch.Character) error {
		c.SetSkill(skill.skillType, level)
		return nil
	}, fmt.Sprintf("set skill %s=%v", skill.name, level))
}

type setEnemyCmd struct {
	Name  string  `arg:"" name:"name" help:"Enemy stat name."`
	Value float32 `arg:"" name:"value" help:"Enemy stat value."`
}

func (cmd *setEnemyCmd) Run(r *editRunner) error {
	name, err := parseStatName(cmd.Name)
	if err != nil {
		return err
	}
	value, err := parseStatValue(cmd.Value)
	if err != nil {
		return err
	}
	return r.apply(func(c *fch.Character) error {
		c.UpsertEnemyStat(name, value)
		return nil
	}, fmt.Sprintf("set enemy %s=%v", name, value))
}

type setMaterialCmd struct {
	Name  string  `arg:"" name:"name" help:"Material stat name."`
	Value float32 `arg:"" name:"value" help:"Material stat value."`
}

func (cmd *setMaterialCmd) Run(r *editRunner) error {
	name, err := parseStatName(cmd.Name)
	if err != nil {
		return err
	}
	value, err := parseStatValue(cmd.Value)
	if err != nil {
		return err
	}
	return r.apply(func(c *fch.Character) error {
		c.UpsertMaterialStat(name, value)
		return nil
	}, fmt.Sprintf("set material %s=%v", name, value))
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
	value, err := parseStatValue(cmd.Value)
	if err != nil {
		return err
	}
	return r.apply(func(c *fch.Character) error {
		return c.SetPlayerStat(stat.index, stat.name, value)
	}, fmt.Sprintf("set player-stat %s=%v", stat.name, value))
}

type listCmd struct {
	Skills     listSkillsCmd     `cmd:"" help:"List known skill names."`
	PlayerStat listPlayerStatCmd `cmd:"" name:"player-stats" help:"List known player stat names."`
	Inventory  listInventoryCmd  `cmd:"" help:"List inventory items in the character file."`
}

type listSkillsCmd struct{}

func (cmd *listSkillsCmd) Run(r *editRunner) error {
	for _, name := range fch.SkillNames() {
		fmt.Fprintln(r.stdout, name)
	}
	return nil
}

type listPlayerStatCmd struct{}

func (cmd *listPlayerStatCmd) Run(r *editRunner) error {
	w := tabwriter.NewWriter(r.stdout, 0, 0, 2, ' ', 0)
	for i, name := range fch.PlayerStatNames() {
		fmt.Fprintf(w, "%d\t%s\n", i, name)
	}
	return w.Flush()
}

type listInventoryCmd struct{}

func (cmd *listInventoryCmd) Run(r *editRunner) error {
	character, err := r.readCharacter()
	if err != nil {
		return err
	}
	return r.listInventory(character)
}

type editRunner struct {
	path     string
	out      string
	dryRun   bool
	noBackup bool
	stdout   io.Writer
}

func (r *editRunner) listInventory(character *fch.Character) error {
	w := tabwriter.NewWriter(r.stdout, 0, 0, 2, ' ', 0)
	for _, item := range character.Player.Inventory {
		fmt.Fprintf(w, "%s\tstack=%d\tquality=%d\tdurability=%v\tgrid=%d,%d\n",
			item.Name,
			item.Stack,
			item.Quality,
			item.Durability,
			item.GridX,
			item.GridY,
		)
	}
	return w.Flush()
}

func (r *editRunner) readCharacter() (*fch.Character, error) {
	if r.path == "" {
		return nil, fmt.Errorf("missing character: set --character or CHARACTER")
	}
	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil, fmt.Errorf("read character %s: %w", r.path, err)
	}
	character, err := fch.DecodeBytes(data)
	if err != nil {
		return nil, fmt.Errorf("decode character %s: %w", r.path, err)
	}
	return character, nil
}

func (r *editRunner) apply(edit func(*fch.Character) error, summary string) error {
	character, err := r.readCharacter()
	if err != nil {
		return err
	}
	if err := edit(character); err != nil {
		return err
	}
	encoded, err := fch.EncodeBytes(character)
	if err != nil {
		return fmt.Errorf("encode character %s: %w", r.path, err)
	}

	target := r.out
	if target == "" {
		target = r.path
	}
	inPlace := target == r.path
	if r.dryRun {
		fmt.Fprintf(r.stdout, "would %s\n", summary)
		fmt.Fprintf(r.stdout, "would write %s\n", target)
		return nil
	}

	if inPlace && !r.noBackup {
		backup, err := backupFile(r.path)
		if err != nil {
			return err
		}
		fmt.Fprintf(r.stdout, "backup %s\n", backup)
	}
	if err := writeFile(target, encoded, r.path); err != nil {
		return fmt.Errorf("write edited character %s: %w", target, err)
	}
	fmt.Fprintln(r.stdout, summary)
	fmt.Fprintf(r.stdout, "wrote %s\n", target)
	return nil
}

func backupFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read backup source %s: %w", path, err)
	}
	backup, err := nextBackupPath(path)
	if err != nil {
		return "", err
	}
	if err := writeFile(backup, data, path); err != nil {
		return "", fmt.Errorf("write backup %s: %w", backup, err)
	}
	return backup, nil
}

func nextBackupPath(path string) (string, error) {
	for i := 0; ; i++ {
		candidate := path + ".bak"
		if i > 0 {
			candidate = fmt.Sprintf("%s.bak.%d", path, i)
		}
		if _, err := os.Stat(candidate); err != nil {
			if os.IsNotExist(err) {
				return candidate, nil
			}
			return "", fmt.Errorf("inspect backup path %s: %w", candidate, err)
		}
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
		kong.Description(`Edit a Valheim character file.

Examples:
  fchedit --character character.fch set skill Run 50
  fchedit --character character.fch --dry-run set player-stat Deaths 0
  fchedit --character character.fch add inventory 'Wood,stack=50,quality=1'
  fchedit list skills
  fchedit list player-stats`),
		kong.Writers(stdout, stderr),
	)
	if err != nil {
		return err
	}
	ctx, err := parser.Parse(args)
	if err != nil {
		return err
	}
	runner := &editRunner{
		path:     strings.TrimSpace(cli.Character),
		out:      strings.TrimSpace(cli.Out),
		dryRun:   cli.DryRun,
		noBackup: cli.NoBackup,
		stdout:   stdout,
	}
	return ctx.Run(runner)
}
