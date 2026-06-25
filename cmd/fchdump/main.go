package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	fch "github.com/lanchelms/fch-decoder"
)

type cli struct {
	Character string `name:"character" env:"CHARACTER" required:"" type:"path" help:"Character file to dump."`
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	var cli cli
	parser, err := kong.New(&cli, kong.Name("fchdump"), kong.Writers(stdout, stderr))
	if err != nil {
		return err
	}
	if _, err := parser.Parse(args); err != nil {
		return err
	}

	f, err := os.Open(cli.Character)
	if err != nil {
		return err
	}
	defer f.Close()

	character, err := fch.Decode(f)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(character); err != nil {
		return err
	}
	return nil
}
