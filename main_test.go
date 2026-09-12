package main

import (
	"bytes"
	"os"
	"path/filepath"
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
	var buf bytes.Buffer
	printText(&buf, &checker.Result{}, ".env.example", ".env", false, false, "")
	if !strings.Contains(buf.String(), "has all keys from") {
		t.Errorf("expected clean message, got: %s", buf.String())
	}
}

func TestPrintText_AllCleanWithScan(t *testing.T) {
	var buf bytes.Buffer
	printText(&buf, &checker.Result{}, ".env.example", ".env", false, false, "./src")
	if !strings.Contains(buf.String(), "no undeclared keys found in ./src") {
		t.Errorf("expected scan clean line, got: %s", buf.String())
	}
}

func TestPrintText_MissingKeys(t *testing.T) {
	r := &checker.Result{Missing: []string{"FOO", "BAR"}}
	var buf bytes.Buffer
	printText(&buf, r, ".env.example", ".env", false, false, "")
	for _, want := range []string{"FOO", "BAR", "missing 2 key(s)"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("expected %q in output, got: %s", want, buf.String())
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
	var buf bytes.Buffer
	printText(&buf, r, ".env.example", ".env", true, true, "./src")
	for _, want := range []string{"M_KEY", "E_KEY", "X_KEY", "U_KEY", "src/foo.go:10"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("expected %q in output, got: %s", want, buf.String())
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
	var buf bytes.Buffer
	printGitHub(&buf, r, ".env.example", ".env", true, true)
	wants := []string{
		`::error file=.env::Missing key "FOO"`,
		`::error file=.env::Required key "BAR" is empty`,
		`::warning file=.env::Extra key "BAZ"`,
		`::error file=src/main.go,line=5::Key "QUX"`,
		`::error file=src/util.go,line=3::Key "QUX"`,
	}
	for _, want := range wants {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("expected %q in output, got: %s", want, buf.String())
		}
	}
}

func TestPrintGitHub_UndeclaredWithoutLineNumber(t *testing.T) {
	r := &checker.Result{
		Undeclared: []checker.UndeclaredKey{
			{Key: "QUX", CallSites: []string{"bare-path-no-line"}},
		},
	}
	var buf bytes.Buffer
	printGitHub(&buf, r, ".env.example", ".env", false, false)
	want := `::error file=bare-path-no-line::Key "QUX"`
	if !strings.Contains(buf.String(), want) {
		t.Errorf("expected %q in output, got: %s", want, buf.String())
	}
}

// run() integration tests — exit code is the contract; assert on it primarily,
// and on stdout/stderr only where the test name implies an output check.

func TestRun_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "envcheck ") {
		t.Errorf("expected version line in stdout, got: %s", stdout.String())
	}
}

func TestRun_BadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--no-such-flag"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("expected exit 2 on bad flag, got %d", code)
	}
}

func TestRun_MissingTemplateFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(
		[]string{"--template", "/no/such/template", "--actual", "/no/such/actual"},
		&stdout, &stderr,
	)
	if code != 2 {
		t.Errorf("expected exit 2 on unreadable files, got %d", code)
	}
	if !strings.Contains(stderr.String(), "envcheck:") {
		t.Errorf("expected error prefix in stderr, got: %s", stderr.String())
	}
}

func TestRun_AllKeysPresent(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, ".env.example", "DB_HOST=\nDB_PORT=\n")
	mustWrite(t, dir, ".env", "DB_HOST=localhost\nDB_PORT=5432\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--template", filepath.Join(dir, ".env.example"),
		"--actual", filepath.Join(dir, ".env"),
		"--ci",
	}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("expected exit 0, got %d (stderr=%s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "has all keys from") {
		t.Errorf("expected clean message, got: %s", stdout.String())
	}
}

func TestRun_MissingKeyWithCI(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, ".env.example", "DB_HOST=\nAPI_KEY=\n")
	mustWrite(t, dir, ".env", "DB_HOST=localhost\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--template", filepath.Join(dir, ".env.example"),
		"--actual", filepath.Join(dir, ".env"),
		"--ci",
	}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit 1 on missing keys with --ci, got %d", code)
	}
}

func TestRun_MissingKeyWithoutCIExitsZero(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, ".env.example", "DB_HOST=\nAPI_KEY=\n")
	mustWrite(t, dir, ".env", "DB_HOST=localhost\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--template", filepath.Join(dir, ".env.example"),
		"--actual", filepath.Join(dir, ".env"),
	}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("expected exit 0 without --ci even when keys missing, got %d", code)
	}
}

func TestRun_RequireValuesFlagsEmpty(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, ".env.example", "DB_HOST=\nAPI_KEY=\n")
	mustWrite(t, dir, ".env", "DB_HOST=localhost\nAPI_KEY=\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--template", filepath.Join(dir, ".env.example"),
		"--actual", filepath.Join(dir, ".env"),
		"--ci",
		"--require-values",
	}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit 1 on empty required value with --ci --require-values, got %d", code)
	}
}

func TestRun_StrictFlagsExtra(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, ".env.example", "DB_HOST=\n")
	mustWrite(t, dir, ".env", "DB_HOST=localhost\nUNKNOWN=x\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--template", filepath.Join(dir, ".env.example"),
		"--actual", filepath.Join(dir, ".env"),
		"--ci",
		"--strict",
	}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit 1 on extra key with --ci --strict, got %d", code)
	}
}

func TestRun_ScanFlagsUndeclared(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, ".env.example", "DB_HOST=\n")
	mustWrite(t, dir, ".env", "DB_HOST=localhost\n")
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, srcDir, "main.go", `package main
import "os"
var _ = os.Getenv("STRIPE_KEY")
`)

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--template", filepath.Join(dir, ".env.example"),
		"--actual", filepath.Join(dir, ".env"),
		"--ci",
		"--scan", srcDir,
	}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit 1 on undeclared scan hit with --ci, got %d", code)
	}
	if !strings.Contains(stdout.String(), "STRIPE_KEY") {
		t.Errorf("expected STRIPE_KEY in output, got: %s", stdout.String())
	}
}

func TestRun_GithubFormat(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, ".env.example", "DB_HOST=\nAPI_KEY=\n")
	mustWrite(t, dir, ".env", "DB_HOST=localhost\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--template", filepath.Join(dir, ".env.example"),
		"--actual", filepath.Join(dir, ".env"),
		"--ci",
		"--format=github",
	}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stdout.String(), `::error file=`) {
		t.Errorf("expected github annotation in stdout, got: %s", stdout.String())
	}
}

func mustWrite(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestStringList_SetSplitsCommasAndTrimsBlanks(t *testing.T) {
	var got stringList
	if err := got.Set("PATH, HOME ,,"); err != nil {
		t.Fatal(err)
	}
	if err := got.Set("AWS_*"); err != nil {
		t.Fatal(err)
	}
	want := []string{"PATH", "HOME", "AWS_*"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if got.String() != "PATH,HOME,AWS_*" {
		t.Errorf("unexpected String(): %q", got.String())
	}
}

func TestPrintText_IgnoredCountLine(t *testing.T) {
	var buf bytes.Buffer
	printText(&buf, &checker.Result{Ignored: []string{"PATH", "HOME"}}, ".env.example", ".env", false, false, "")
	if !strings.Contains(buf.String(), "ignored 2 key(s) via --ignore") {
		t.Errorf("expected ignored count line, got: %s", buf.String())
	}
}

func TestPrintText_NoIgnoredLineWhenNothingSuppressed(t *testing.T) {
	var buf bytes.Buffer
	printText(&buf, &checker.Result{}, ".env.example", ".env", false, false, "")
	if strings.Contains(buf.String(), "--ignore") {
		t.Errorf("did not expect an ignore line, got: %s", buf.String())
	}
}

func TestRun_IgnoredMissingKeyExitsZeroWithCI(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, ".env.example", "DB_HOST=\nLEGACY_TOKEN=\n")
	mustWrite(t, dir, ".env", "DB_HOST=localhost\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--template", filepath.Join(dir, ".env.example"),
		"--actual", filepath.Join(dir, ".env"),
		"--ci",
		"--ignore", "LEGACY_TOKEN",
	}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("expected exit 0 when the only missing key is ignored, got %d (stdout=%s)", code, stdout.String())
	}
	if strings.Contains(stdout.String(), "LEGACY_TOKEN") {
		t.Errorf("ignored key should not be reported, got: %s", stdout.String())
	}
}

func TestRun_IgnoreRepeatedFlagMatchesCommaList(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, ".env.example", "DB_HOST=\nA_KEY=\nB_KEY=\n")
	mustWrite(t, dir, ".env", "DB_HOST=localhost\n")
	base := []string{
		"--template", filepath.Join(dir, ".env.example"),
		"--actual", filepath.Join(dir, ".env"),
		"--ci",
	}

	var repeatedOut, commaOut bytes.Buffer
	var stderr bytes.Buffer
	repeated := run(append(append([]string{}, base...), "--ignore", "A_KEY", "--ignore", "B_KEY"), &repeatedOut, &stderr)
	comma := run(append(append([]string{}, base...), "--ignore", "A_KEY,B_KEY"), &commaOut, &stderr)

	if repeated != 0 || comma != 0 {
		t.Errorf("expected both forms to exit 0, got repeated=%d comma=%d", repeated, comma)
	}
	if repeatedOut.String() != commaOut.String() {
		t.Errorf("repeated and comma forms differ:\n repeated=%q\n comma=%q", repeatedOut.String(), commaOut.String())
	}
}

func TestRun_IgnoreOnlySuppressesMatchingKeys(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, ".env.example", "AWS_REGION=\nAWS_KEY=\nSTRIPE_KEY=\n")
	mustWrite(t, dir, ".env", "")

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--template", filepath.Join(dir, ".env.example"),
		"--actual", filepath.Join(dir, ".env"),
		"--ci",
		"--ignore", "AWS_*",
	}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit 1, STRIPE_KEY is still missing, got %d", code)
	}
	if !strings.Contains(stdout.String(), "STRIPE_KEY") {
		t.Errorf("expected STRIPE_KEY reported, got: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "AWS_") {
		t.Errorf("AWS_* keys should be suppressed, got: %s", stdout.String())
	}
}

func TestRun_InvalidIgnorePatternExitsTwo(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, ".env.example", "DB_HOST=\n")
	mustWrite(t, dir, ".env", "DB_HOST=x\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--template", filepath.Join(dir, ".env.example"),
		"--actual", filepath.Join(dir, ".env"),
		"--ignore", "AWS_[",
	}, &stdout, &stderr)

	if code != 2 {
		t.Errorf("expected exit 2 on malformed ignore pattern, got %d", code)
	}
	if !strings.Contains(stderr.String(), "invalid --ignore pattern") {
		t.Errorf("expected a helpful stderr message, got: %s", stderr.String())
	}
}
