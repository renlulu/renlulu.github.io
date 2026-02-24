package backend_test

import (
	"testing"

	backend "github.com/renlulu/hearx/backend"
	"github.com/renlulu/hearx/backend/internal/skillpack"
)

func TestLoadSkillPacks(t *testing.T) {
	loader, err := skillpack.NewLoader(backend.SkillsFS)
	if err != nil {
		t.Fatalf("failed to load skill packs: %v", err)
	}

	packs := loader.List()
	if len(packs) < 2 {
		t.Fatalf("expected at least 2 skill packs, got %d", len(packs))
	}

	// Check general pack exists
	gen, ok := loader.Get("general")
	if !ok {
		t.Fatal("general skill pack not found")
	}
	if gen.Name != "general" {
		t.Errorf("expected name 'general', got %q", gen.Name)
	}
	if len(gen.Glossary) == 0 {
		t.Error("general glossary should not be empty")
	}

	// Check crypto pack exists
	crypto, ok := loader.Get("crypto-web3")
	if !ok {
		t.Fatal("crypto-web3 skill pack not found")
	}
	if len(crypto.Entities) == 0 {
		t.Error("crypto-web3 should have entity rules")
	}
}

func TestMergeSkillPacks(t *testing.T) {
	loader, err := skillpack.NewLoader(backend.SkillsFS)
	if err != nil {
		t.Fatalf("failed to load skill packs: %v", err)
	}

	merged, err := loader.Merge([]string{"crypto-web3"})
	if err != nil {
		t.Fatalf("failed to merge skill packs: %v", err)
	}

	// Should have glossary entries from both general and crypto
	if _, ok := merged.Glossary["API"]; !ok {
		t.Error("merged glossary should contain general term 'API'")
	}
	if _, ok := merged.Glossary["blockchain"]; !ok {
		t.Error("merged glossary should contain crypto term 'blockchain'")
	}

	// Should have prompts from both packs
	if len(merged.Prompts) < 2 {
		t.Errorf("expected at least 2 prompts, got %d", len(merged.Prompts))
	}

	// Should have entity rules from crypto
	if len(merged.Entities) == 0 {
		t.Error("merged should have entity rules from crypto pack")
	}
}

func TestMergeUnknownPack(t *testing.T) {
	loader, err := skillpack.NewLoader(backend.SkillsFS)
	if err != nil {
		t.Fatalf("failed to load skill packs: %v", err)
	}

	_, err = loader.Merge([]string{"nonexistent"})
	if err == nil {
		t.Error("expected error for unknown skill pack")
	}
}
