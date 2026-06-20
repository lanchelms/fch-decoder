package main

import (
	"encoding/json"
	"fmt"
	"os"

	fch "github.com/brian/fch-decoder"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <character.fch>\n", os.Args[0])
		os.Exit(2)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	character, err := fch.Decode(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(character); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
