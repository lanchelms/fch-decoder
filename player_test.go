package fch

import (
	"testing"
	"time"
)

func TestNewPlayer(t *testing.T) {
	before := time.Now().Unix()
	player := NewPlayer("New Player", 654321)
	after := time.Now().Unix()

	if player.Name != "New Player" || player.PlayerID != 654321 {
		t.Fatalf("player = %q/%d, want New Player/654321", player.Name, player.PlayerID)
	}
	if player.DateCreatedUnix < before || player.DateCreatedUnix > after {
		t.Fatalf("DateCreatedUnix = %d, want between %d and %d", player.DateCreatedUnix, before, after)
	}
	if player.PlayerVersion != supportedPlayerVersion {
		t.Fatalf("PlayerVersion = %d, want %d", player.PlayerVersion, supportedPlayerVersion)
	}
	if player.InventoryVersion != supportedInventoryVersion {
		t.Fatalf("InventoryVersion = %d, want %d", player.InventoryVersion, supportedInventoryVersion)
	}
	if player.SkillVersion != supportedSkillVersion {
		t.Fatalf("SkillVersion = %d, want %d", player.SkillVersion, supportedSkillVersion)
	}
}
