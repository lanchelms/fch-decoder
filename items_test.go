package fch

import (
	"slices"
	"testing"
)

func TestItemMetadataLookup(t *testing.T) {
	item, ok := Items().Lookup("SwordIron")
	if !ok {
		t.Fatal("Items().Lookup(SwordIron) ok = false")
	}
	if !item.InventoryValid || !slices.Equal(item.Recipes, []string{"Recipe_SwordIron"}) || item.BaseDurability != 200 || item.DurabilityStep != 50 || item.MaxQuality != 4 || item.MaxStack != 1 {
		t.Fatalf("SwordIron metadata = %+v", item)
	}
	if got := item.Durability(3); got != 300 {
		t.Fatalf("SwordIron durability quality 3 = %v, want 300", got)
	}
}

func TestItemMetadataInventoryValid(t *testing.T) {
	item, ok := Items().Lookup("Abomination_attack1")
	if !ok {
		t.Fatal("Items().Lookup(Abomination_attack1) ok = false")
	}
	if item.InventoryValid {
		t.Fatalf("Abomination_attack1 InventoryValid = true, want false")
	}
	if len(item.Recipes) != 0 {
		t.Fatalf("Abomination_attack1 Recipes = %v, want none", item.Recipes)
	}
}

func TestItemMetadataRecipes(t *testing.T) {
	item, ok := Items().Lookup("Bronze")
	if !ok {
		t.Fatal("Items().Lookup(Bronze) ok = false")
	}
	if !slices.Equal(item.Recipes, []string{"Recipe_Bronze", "Recipe_Bronze5"}) {
		t.Fatalf("Bronze recipes = %v, want both bronze recipes", item.Recipes)
	}
}

func TestItemMetadataList(t *testing.T) {
	items := Items().List()
	names := Items().Names()
	if len(items) == 0 {
		t.Fatal("Items().List returned no items")
	}
	if len(names) != len(items) {
		t.Fatalf("Items().Names length = %d, want %d", len(names), len(items))
	}
	if names[0] != items[0].Name {
		t.Fatalf("Items().Names[0] = %q, want %q", names[0], items[0].Name)
	}
}
