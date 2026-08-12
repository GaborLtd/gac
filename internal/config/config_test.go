package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadYAMLWithPromptTemplate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gac.yaml")
	want := Config{
		Provider:       "agy",
		Model:          "cheap-model",
		Language:       "en",
		DiffMaxBytes:   1234,
		DiffMaxLines:   12,
		PromptTemplate: "Use {{.Language}}.\nDiff:\n{{.Diff}}",
		TimeoutSeconds: 9,
		SkipCIMode:     "ask",
	}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider != want.Provider || got.Model != want.Model || got.Language != want.Language || got.DiffMaxBytes != want.DiffMaxBytes || got.DiffMaxLines != want.DiffMaxLines || got.TimeoutSeconds != want.TimeoutSeconds || got.SkipCIMode != want.SkipCIMode || strings.TrimSpace(got.PromptTemplate) != want.PromptTemplate {
		t.Fatalf("config round-trip mismatch: got %#v want %#v", got, want)
	}
}

func TestLoadMissingUsesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Language != "en" || cfg.DiffMaxBytes <= 0 || cfg.DiffMaxLines <= 0 {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("unknown: value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected unknown field error")
	}
}
