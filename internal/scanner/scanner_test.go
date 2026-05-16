package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestScan_Go(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", `package main
import "os"
func main() {
	_ = os.Getenv("DB_HOST")
	if v, ok := os.LookupEnv("API_KEY"); ok { _ = v }
}
`)
	r := scanOK(t, dir)
	assertKeys(t, r, []string{"API_KEY", "DB_HOST"})
}

func TestScan_NodeAndTS(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "server.ts", `
const port = process.env.PORT;
const url = process.env["DATABASE_URL"];
const v = import.meta.env.VITE_API;
`)
	r := scanOK(t, dir)
	assertKeys(t, r, []string{"DATABASE_URL", "PORT", "VITE_API"})
}

func TestScan_Python(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "app.py", `import os
a = os.environ["SECRET_KEY"]
b = os.environ.get("DEBUG")
c = os.getenv("REDIS_URL", "redis://localhost")
`)
	r := scanOK(t, dir)
	assertKeys(t, r, []string{"DEBUG", "REDIS_URL", "SECRET_KEY"})
}

func TestScan_IgnoresVendorAndNodeModules(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "app.go", `package main
import "os"
var _ = os.Getenv("REAL_KEY")
`)
	write(t, filepath.Join(dir, "node_modules", "pkg"), "leak.js", `process.env.STRAY_FROM_NODE_MODULES`)
	write(t, filepath.Join(dir, "vendor", "lib"), "v.go", `var _ = os.Getenv("STRAY_FROM_VENDOR")`)

	r := scanOK(t, dir)
	assertKeys(t, r, []string{"REAL_KEY"})
}

func TestScan_TracksCallSites(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", `package main
import "os"
var x = os.Getenv("SHARED")
`)
	write(t, dir, "b.go", `package main
import "os"
var y = os.Getenv("SHARED")
`)
	r := scanOK(t, dir)
	locs := r.UsedKeys["SHARED"]
	if len(locs) != 2 {
		t.Fatalf("expected 2 call sites for SHARED, got %v", locs)
	}
}

func TestScan_IgnoresUnknownExtensions(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "config.toml", `key = "process.env.NOT_PARSED"`)
	r := scanOK(t, dir)
	if len(r.UsedKeys) != 0 {
		t.Errorf("expected no keys from .toml, got %v", r.UsedKeys)
	}
}

func TestScan_NonexistentPath(t *testing.T) {
	if _, err := Scan(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Error("expected error for missing path")
	}
}

func scanOK(t *testing.T, dir string) *Result {
	t.Helper()
	r, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func assertKeys(t *testing.T, r *Result, want []string) {
	t.Helper()
	got := make([]string, 0, len(r.UsedKeys))
	for k := range r.UsedKeys {
		got = append(got, k)
	}
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("keys mismatch: want %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("keys mismatch: want %v, got %v", want, got)
		}
	}
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
