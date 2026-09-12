package checker

import (
	"os"
	"path/filepath"
	"strings"
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

func TestCheck_ScanFindsUndeclared(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DB_HOST=\n")
	writeFile(t, dir, ".env", "DB_HOST=localhost\n")
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, srcDir, "main.go", `package main
import "os"
var _ = os.Getenv("DB_HOST")
var _ = os.Getenv("STRIPE_KEY")
`)
	r, err := Check(
		filepath.Join(dir, ".env.example"),
		filepath.Join(dir, ".env"),
		Options{ScanPath: srcDir},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Undeclared) != 1 || r.Undeclared[0].Key != "STRIPE_KEY" {
		t.Fatalf("expected STRIPE_KEY undeclared, got %+v", r.Undeclared)
	}
	if len(r.Undeclared[0].CallSites) == 0 {
		t.Errorf("expected call sites to be populated")
	}
}

func TestCheck_IgnoreSuppressesAllCategories(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DB_HOST=\nLEGACY_TOKEN=\nEMPTY_ONE=\n")
	writeFile(t, dir, ".env", "DB_HOST=localhost\nEMPTY_ONE=\nSTALE_KEY=y\n")
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, srcDir, "main.go", `package main
import "os"
var _ = os.Getenv("SCANNED_KEY")
`)

	opts := Options{
		RequireValues: true,
		ScanPath:      srcDir,
		Ignore:        []string{"LEGACY_TOKEN", "EMPTY_ONE", "STALE_KEY", "SCANNED_KEY"},
	}
	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Missing) != 0 {
		t.Errorf("expected no missing, got %v", r.Missing)
	}
	if len(r.Empty) != 0 {
		t.Errorf("expected no empty, got %v", r.Empty)
	}
	if len(r.Extra) != 0 {
		t.Errorf("expected no extra, got %v", r.Extra)
	}
	if len(r.Undeclared) != 0 {
		t.Errorf("expected no undeclared, got %+v", r.Undeclared)
	}
	if len(r.Ignored) != 4 {
		t.Errorf("expected 4 suppressed keys, got %v", r.Ignored)
	}
}

func TestCheck_IgnoreGlobPatterns(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DB_HOST=\n")
	writeFile(t, dir, ".env", "DB_HOST=x\nAWS_REGION=1\nAWS_KEY=2\nMY_SECRET=3\nPORT=4\nKEEP_ME=5\n")

	opts := Options{Ignore: []string{"AWS_*", "*_SECRET", "P?RT"}}
	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Extra) != 1 || r.Extra[0] != "KEEP_ME" {
		t.Errorf("expected only KEEP_ME extra, got %v", r.Extra)
	}
}

func TestCheck_IgnoreInvalidPatternReturnsError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DB_HOST=\n")
	writeFile(t, dir, ".env", "DB_HOST=x\n")

	_, err := Check(
		filepath.Join(dir, ".env.example"),
		filepath.Join(dir, ".env"),
		Options{Ignore: []string{"AWS_["}},
	)
	if err == nil {
		t.Fatal("expected error for malformed ignore pattern, got nil")
	}
	if !strings.Contains(err.Error(), "AWS_[") {
		t.Errorf("expected the bad pattern in the error message, got: %v", err)
	}
}

func TestCheck_IgnoreNonMatchingPatternChangesNothing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DB_HOST=\nAPI_KEY=\n")
	writeFile(t, dir, ".env", "DB_HOST=x\n")

	opts := Options{Ignore: []string{"NOPE_*"}}
	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Missing) != 1 || r.Missing[0] != "API_KEY" {
		t.Errorf("expected API_KEY still missing, got %v", r.Missing)
	}
	if len(r.Ignored) != 0 {
		t.Errorf("expected nothing suppressed, got %v", r.Ignored)
	}
}

func TestCheck_IgnoreDoesNotCountKeysThatWereNeverReported(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DB_HOST=\n")
	writeFile(t, dir, ".env", "DB_HOST=localhost\n")

	opts := Options{Ignore: []string{"DB_HOST"}}
	r, err := Check(filepath.Join(dir, ".env.example"), filepath.Join(dir, ".env"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Ignored) != 0 {
		t.Errorf("a healthy key should not be counted as suppressed, got %v", r.Ignored)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
