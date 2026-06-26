package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestRunDumpsCharacter(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"--character", filepath.Join("..", "..", "testdata", "Steam_333333_tugen.fch")}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run error = %v, stderr = %s", err, stderr.String())
	}

	var out map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["player"] == nil {
		t.Fatalf("output has no player: %s", stdout.String())
	}
}

func TestRunDumpsCharacterShortFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"-c", filepath.Join("..", "..", "testdata", "Steam_333333_tugen.fch")}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run error = %v, stderr = %s", err, stderr.String())
	}

	var out map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["player"] == nil {
		t.Fatalf("output has no player: %s", stdout.String())
	}
}

func TestRunUsesCharacterEnv(t *testing.T) {
	t.Setenv("CHARACTER", filepath.Join("..", "..", "testdata", "Steam_333333_tugen.fch"))

	var stdout, stderr bytes.Buffer
	err := run(nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run error = %v, stderr = %s", err, stderr.String())
	}

	var out map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["player"] == nil {
		t.Fatalf("output has no player: %s", stdout.String())
	}
}

func TestRunRequiresCharacter(t *testing.T) {
	err := run(nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("run error = nil, want missing character")
	}
}
