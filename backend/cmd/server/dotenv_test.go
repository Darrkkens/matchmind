package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvFromParentWithoutOverriding(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "backend")
	_ = os.Mkdir(sub, 0o755)
	_ = os.WriteFile(filepath.Join(root, ".env"), []byte("# comment\nMM_A=from-file\nexport MM_B=\"quoted\"\nMM_C=file\nbroken line\nMM_URL=postgres://u:p@h:5/db?sslmode=disable\n"), 0o600)
	t.Chdir(sub)
	t.Setenv("MM_C", "from-shell")
	for _, k := range []string{"MM_A", "MM_B", "MM_URL"} {
		_ = os.Unsetenv(k)
		t.Cleanup(func() { _ = os.Unsetenv(k) })
	}
	if got := loadDotEnv(); got != "../.env" {
		t.Fatalf("loaded %q", got)
	}
	if os.Getenv("MM_A") != "from-file" || os.Getenv("MM_B") != "quoted" || os.Getenv("MM_C") != "from-shell" || os.Getenv("MM_URL") != "postgres://u:p@h:5/db?sslmode=disable" {
		t.Fatalf("A=%q B=%q C=%q URL=%q", os.Getenv("MM_A"), os.Getenv("MM_B"), os.Getenv("MM_C"), os.Getenv("MM_URL"))
	}
}
