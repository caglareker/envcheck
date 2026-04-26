package checker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheck_AllPresent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DB_HOST=\nDB_PORT=\nAPI_KEY=\n")
	writeFile(t, dir, ".env", "DB_HOST=localhost\nDB_PORT=5432\nAPI_KEY=secret\n")

	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Missing) != 0 {
		t.Errorf("expected no missing, got %v", r.Missing)
	}
}

func TestCheck_SomeMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DB_HOST=\nDB_PORT=\nAPI_KEY=\n")
	writeFile(t, dir, ".env", "DB_HOST=localhost\n")

	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Missing) != 2 {
		t.Errorf("expected 2 missing, got %v", r.Missing)
	}
}

func TestCheck_IgnoresComments(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "# this is a comment\nDB_HOST=\n\n# another comment\nDB_PORT=\n")
	writeFile(t, dir, ".env", "DB_HOST=x\nDB_PORT=1\n")

	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Missing) != 0 {
		t.Errorf("expected no missing, got %v", r.Missing)
	}
}

func TestCheck_ReportsExtraKeys(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DB_HOST=\nDB_PORT=\n")
	writeFile(t, dir, ".env", "DB_HOST=localhost\nDB_PORT=5432\nLEGACY_KEY=x\nUNUSED=y\n")

	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Missing) != 0 {
		t.Errorf("expected no missing, got %v", r.Missing)
	}
	if len(r.Extra) != 2 {
		t.Errorf("expected 2 extra, got %v", r.Extra)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
