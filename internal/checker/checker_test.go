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

	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"), Options{})
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

	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"), Options{})
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

	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"), Options{})
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

	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"), Options{})
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

func TestCheck_RequireValues_DetectsEmpty(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DB_HOST=\nDB_PORT=\nAPI_KEY=\n")
	writeFile(t, dir, ".env", "DB_HOST=localhost\nDB_PORT=\nAPI_KEY=secret\n")

	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"), Options{RequireValues: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Missing) != 0 {
		t.Errorf("expected no missing, got %v", r.Missing)
	}
	if len(r.Empty) != 1 || r.Empty[0] != "DB_PORT" {
		t.Errorf("expected DB_PORT in Empty, got %v", r.Empty)
	}
}

func TestCheck_RequireValues_OffByDefault(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DB_HOST=\n")
	writeFile(t, dir, ".env", "DB_HOST=\n")

	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Empty) != 0 {
		t.Errorf("expected no Empty when RequireValues disabled, got %v", r.Empty)
	}
}

func TestCheck_HandlesExportPrefix(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "export DB_HOST=\nexport DB_PORT=\n")
	writeFile(t, dir, ".env", "export DB_HOST=localhost\nexport DB_PORT=5432\n")

	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Missing) != 0 || len(r.Extra) != 0 {
		t.Errorf("expected clean result, got missing=%v extra=%v", r.Missing, r.Extra)
	}
}

func TestCheck_HandlesQuotedValues(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "GREETING=\nMOTD=\n")
	writeFile(t, dir, ".env", "GREETING=\"hello world\"\nMOTD='multi word'\n")

	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"), Options{RequireValues: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Missing) != 0 {
		t.Errorf("expected no missing, got %v", r.Missing)
	}
	if len(r.Empty) != 0 {
		t.Errorf("quoted values should count as non-empty, got Empty=%v", r.Empty)
	}
}

func TestCheck_EmptyQuotedValueIsEmpty(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "TOKEN=\n")
	writeFile(t, dir, ".env", "TOKEN=\"\"\n")

	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"), Options{RequireValues: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Empty) != 1 || r.Empty[0] != "TOKEN" {
		t.Errorf("expected TOKEN in Empty (\"\" unwraps to empty), got %v", r.Empty)
	}
}

func TestCheck_StripsInlineComment(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "PORT=\n")
	writeFile(t, dir, ".env", "PORT=5432 # the database port\n")

	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"), Options{RequireValues: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Empty) != 0 {
		t.Errorf("inline comment should not blank out value, got Empty=%v", r.Empty)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
