package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/caglareker/envcheck/internal/checker"
)

func TestSplitCallSite(t *testing.T) {
	cases := []struct {
		in       string
		wantFile string
		wantLine string
	}{
		{"main.go:42", "main.go", "42"},
		{"path/to/file.go:1", "path/to/file.go", "1"},
		{"no-colon", "no-colon", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		f, l := splitCallSite(c.in)
		if f != c.wantFile || l != c.wantLine {
			t.Errorf("splitCallSite(%q): got (%q,%q), want (%q,%q)", c.in, f, l, c.wantFile, c.wantLine)
		}
	}
}

func TestPrintText_AllClean(t *testing.T) {
	out := capture(t, func() {
		printText(&checker.Result{}, ".env.example", ".env", false, false, "")
	})
	if !strings.Contains(out, "has all keys from") {
		t.Errorf("expected clean message, got: %s", out)
	}
}

func TestPrintText_AllCleanWithScan(t *testing.T) {
	out := capture(t, func() {
		printText(&checker.Result{}, ".env.example", ".env", false, false, "./src")
	})
	if !strings.Contains(out, "no undeclared keys found in ./src") {
		t.Errorf("expected scan clean line, got: %s", out)
	}
}

func TestPrintText_MissingKeys(t *testing.T) {
	r := &checker.Result{Missing: []string{"FOO", "BAR"}}
	out := capture(t, func() {
		printText(r, ".env.example", ".env", false, false, "")
	})
	for _, want := range []string{"FOO", "BAR", "missing 2 key(s)"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestPrintText_AllCategories(t *testing.T) {
	r := &checker.Result{
		Missing: []string{"M_KEY"},
		Empty:   []string{"E_KEY"},
		Extra:   []string{"X_KEY"},
		Undeclared: []checker.UndeclaredKey{
			{Key: "U_KEY", CallSites: []string{"src/foo.go:10"}},
		},
	}
	out := capture(t, func() {
		printText(r, ".env.example", ".env", true, true, "./src")
	})
	for _, want := range []string{"M_KEY", "E_KEY", "X_KEY", "U_KEY", "src/foo.go:10"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestPrintGitHub_AllCategories(t *testing.T) {
	r := &checker.Result{
		Missing: []string{"FOO"},
		Empty:   []string{"BAR"},
		Extra:   []string{"BAZ"},
		Undeclared: []checker.UndeclaredKey{
			{Key: "QUX", CallSites: []string{"src/main.go:5", "src/util.go:3"}},
		},
	}
	out := capture(t, func() {
		printGitHub(r, ".env.example", ".env", true, true)
	})
	wants := []string{
		`::error file=.env::Missing key "FOO"`,
		`::error file=.env::Required key "BAR" is empty`,
		`::warning file=.env::Extra key "BAZ"`,
		`::error file=src/main.go,line=5::Key "QUX"`,
		`::error file=src/util.go,line=3::Key "QUX"`,
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestPrintGitHub_UndeclaredWithoutLineNumber(t *testing.T) {
	r := &checker.Result{
		Undeclared: []checker.UndeclaredKey{
			{Key: "QUX", CallSites: []string{"bare-path-no-line"}},
		},
	}
	out := capture(t, func() {
		printGitHub(r, ".env.example", ".env", false, false)
	})
	want := `::error file=bare-path-no-line::Key "QUX"`
	if !strings.Contains(out, want) {
		t.Errorf("expected %q in output, got: %s", want, out)
	}
}

// capture redirects stdout for the duration of fn and returns what was written.
func capture(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
