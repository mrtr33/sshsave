package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func useTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func mode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

func TestSaveUsesOwnerOnlyPermissions(t *testing.T) {
	home := useTempHome(t)
	dir := filepath.Join(home, ".config", "sshlauncher")

	// Simulate a config made by an older version with loose permissions
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sshlauncher.json"), []byte(`{"entries":{}}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := NewConfig()
	if err := cfg.AddEntry("a", "ssh a", nil, false); err != nil {
		t.Fatal(err)
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}

	if m := mode(t, dir); m != 0700 {
		t.Errorf("dir mode = %o, want 700", m)
	}
	for _, f := range []string{"sshlauncher.json", "sshlauncher.json.bak"} {
		if m := mode(t, filepath.Join(dir, f)); m != 0600 {
			t.Errorf("%s mode = %o, want 600", f, m)
		}
	}
}

func TestSaveKeepsBackupAndLeavesNoTempFiles(t *testing.T) {
	home := useTempHome(t)
	dir := filepath.Join(home, ".config", "sshlauncher")

	cfg := NewConfig()
	cfg.AddEntry("first", "ssh first", nil, false)
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	cfg.AddEntry("second", "ssh second", nil, false)
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Entries) != 2 {
		t.Errorf("loaded %d entries, want 2", len(loaded.Entries))
	}

	backup, err := os.ReadFile(filepath.Join(dir, "sshlauncher.json.bak"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(backup), "first") || strings.Contains(string(backup), "second") {
		t.Errorf("backup should hold only the previous config, got %s", backup)
	}

	files, _ := os.ReadDir(dir)
	if len(files) != 2 {
		t.Errorf("expected only config and backup, found %d files", len(files))
	}
}

func TestForceUpdateKeepsSecretsAndAliases(t *testing.T) {
	cfg := NewConfig()
	cfg.AddEntry("work", "ssh old", []string{"w"}, false)
	cfg.Entries["work"].EncryptedSecret = "enc"
	cfg.Entries["work"].UseSecret = true

	if err := cfg.AddEntry("work", "ssh new", nil, true); err != nil {
		t.Fatal(err)
	}

	e := cfg.Entries["work"]
	if e.Command != "ssh new" {
		t.Errorf("command = %q", e.Command)
	}
	if e.EncryptedSecret != "enc" || !e.UseSecret {
		t.Error("secret was lost on --force")
	}
	if len(e.Alias) != 1 || e.Alias[0] != "w" {
		t.Errorf("aliases = %v, want [w]", e.Alias)
	}

	if err := cfg.AddEntry("work", "ssh new", []string{"x"}, true); err != nil {
		t.Fatal(err)
	}
	if len(e.Alias) != 1 || e.Alias[0] != "x" {
		t.Errorf("aliases = %v, want [x]", e.Alias)
	}
}

func TestAddWithoutForceFails(t *testing.T) {
	cfg := NewConfig()
	cfg.AddEntry("a", "ssh a", nil, false)
	if err := cfg.AddEntry("a", "ssh b", nil, false); err == nil {
		t.Error("expected error for existing entry")
	}
}

func TestAliasConflicts(t *testing.T) {
	cfg := NewConfig()
	if err := cfg.AddEntry("work", "ssh w", []string{"w"}, false); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		aliases []string
	}{
		{"other", []string{"w"}},      // alias used by another entry
		{"other", []string{"work"}},   // alias equals another entry's name
		{"w", nil},                    // name equals another entry's alias
		{"other", []string{"x", "x"}}, // duplicate alias
		{"other", []string{"other"}},  // alias equals own name
	}
	for _, c := range cases {
		if err := cfg.AddEntry(c.name, "ssh o", c.aliases, false); err == nil {
			t.Errorf("AddEntry(%q, %v) should fail", c.name, c.aliases)
		}
		delete(cfg.Entries, "other")
	}

	// Re-adding an entry with its own alias is fine
	if err := cfg.AddEntry("work", "ssh w2", []string{"w"}, true); err != nil {
		t.Errorf("updating own alias failed: %v", err)
	}
}

func TestFindAndRemoveByAlias(t *testing.T) {
	cfg := NewConfig()
	cfg.AddEntry("work", "ssh w", []string{"w"}, false)

	_, name, ok := cfg.FindEntry("w")
	if !ok || name != "work" {
		t.Fatalf("FindEntry(w) = %q, %v", name, ok)
	}

	removed, err := cfg.RemoveEntry("w")
	if err != nil || removed != "work" {
		t.Fatalf("RemoveEntry(w) = %q, %v", removed, err)
	}
	if len(cfg.Entries) != 0 {
		t.Error("entry not removed")
	}
	if _, err := cfg.RemoveEntry("w"); err == nil {
		t.Error("expected error removing missing entry")
	}
}

func TestNames(t *testing.T) {
	cfg := NewConfig()
	cfg.AddEntry("b", "ssh b", []string{"bb"}, false)
	cfg.AddEntry("a", "ssh a", nil, false)

	got := cfg.Names()
	want := []string{"a", "b", "bb"}
	if len(got) != len(want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Names() = %v, want %v", got, want)
		}
	}
}

func TestUpdateEntrySetsExactAliases(t *testing.T) {
	cfg := NewConfig()
	cfg.AddEntry("work", "ssh w", []string{"w", "x"}, false)
	cfg.Entries["work"].EncryptedSecret = "enc"

	if err := cfg.UpdateEntry("work", "ssh new", nil); err != nil {
		t.Fatal(err)
	}
	e := cfg.Entries["work"]
	if e.Command != "ssh new" || len(e.Alias) != 0 || e.EncryptedSecret != "enc" {
		t.Errorf("unexpected entry %+v", e)
	}
	if err := cfg.UpdateEntry("missing", "ssh m", nil); err == nil {
		t.Error("expected error for missing entry")
	}
}

func TestRenameEntry(t *testing.T) {
	cfg := NewConfig()
	cfg.AddEntry("work", "ssh w", []string{"w"}, false)
	cfg.AddEntry("home", "ssh h", []string{"h"}, false)

	old, err := cfg.RenameEntry("w", "office")
	if err != nil || old != "work" {
		t.Fatalf("RenameEntry = %q, %v", old, err)
	}
	if _, ok := cfg.Entries["office"]; !ok {
		t.Error("new name missing")
	}
	if _, ok := cfg.Entries["work"]; ok {
		t.Error("old name still present")
	}
	if _, name, _ := cfg.FindEntry("w"); name != "office" {
		t.Errorf("alias now points to %q", name)
	}

	for _, bad := range []string{"home", "h", "w"} {
		if _, err := cfg.RenameEntry("office", bad); err == nil {
			t.Errorf("rename to %q should fail", bad)
		}
		if _, ok := cfg.Entries["office"]; !ok {
			t.Fatalf("failed rename to %q lost the entry", bad)
		}
	}

	if _, err := cfg.RenameEntry("missing", "x"); err == nil {
		t.Error("expected error for missing entry")
	}
}
