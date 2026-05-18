package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/caglareker/envcheck/internal/checker"
)

// Build-time variables injected via -ldflags by goreleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("envcheck", flag.ContinueOnError)
	fs.SetOutput(stderr)
	template := fs.String("template", ".env.example", "template env file with required keys")
	actual := fs.String("actual", ".env", "actual env file to check")
	ci := fs.Bool("ci", false, "exit non-zero when problems are found")
	strict := fs.Bool("strict", false, "also report keys present in actual but not in template")
	requireValues := fs.Bool("require-values", false, "fail when a required key is present but empty in actual")
	scan := fs.String("scan", "", "scan a source directory for env key usage and flag keys missing from the template")
	format := fs.String("format", "text", "output format: text|github")
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintf(stdout, "envcheck %s (commit %s, built %s)\n", version, commit, date)
		return 0
	}

	opts := checker.Options{
		RequireValues: *requireValues,
		ScanPath:      *scan,
	}
	result, err := checker.Check(*template, *actual, opts)
	if err != nil {
		fmt.Fprintf(stderr, "envcheck: %v\n", err)
		return 2
	}

	switch *format {
	case "github":
		printGitHub(stdout, result, *template, *actual, *strict, *requireValues)
	default:
		printText(stdout, result, *template, *actual, *strict, *requireValues, *scan)
	}

	if *ci {
		if len(result.Missing) > 0 {
			return 1
		}
		if *requireValues && len(result.Empty) > 0 {
			return 1
		}
		if *strict && len(result.Extra) > 0 {
			return 1
		}
		if len(result.Undeclared) > 0 {
			return 1
		}
	}
	return 0
}

func printText(w io.Writer, r *checker.Result, template, actual string, strict, requireValues bool, scanPath string) {
	clean := len(r.Missing) == 0 &&
		len(r.Undeclared) == 0 &&
		!(requireValues && len(r.Empty) > 0) &&
		!(strict && len(r.Extra) > 0)
	if clean {
		fmt.Fprintf(w, "%s has all keys from %s\n", actual, template)
		if scanPath != "" {
			fmt.Fprintf(w, "scan: no undeclared keys found in %s\n", scanPath)
		}
		return
	}

	if len(r.Missing) > 0 {
		fmt.Fprintf(w, "%s is missing %d key(s) defined in %s:\n", actual, len(r.Missing), template)
		for _, k := range r.Missing {
			fmt.Fprintf(w, "  - %s\n", k)
		}
	}
	if requireValues && len(r.Empty) > 0 {
		fmt.Fprintf(w, "%s has %d empty value(s) for required key(s):\n", actual, len(r.Empty))
		for _, k := range r.Empty {
			fmt.Fprintf(w, "  ! %s\n", k)
		}
	}
	if strict && len(r.Extra) > 0 {
		fmt.Fprintf(w, "%s has %d extra key(s) not in %s:\n", actual, len(r.Extra), template)
		for _, k := range r.Extra {
			fmt.Fprintf(w, "  + %s\n", k)
		}
	}
	if len(r.Undeclared) > 0 {
		fmt.Fprintf(w, "found %d key(s) used in code but missing from %s:\n", len(r.Undeclared), template)
		for _, u := range r.Undeclared {
			fmt.Fprintf(w, "  ? %s\n", u.Key)
			for _, site := range u.CallSites {
				fmt.Fprintf(w, "      %s\n", site)
			}
		}
	}
}

func printGitHub(w io.Writer, r *checker.Result, template, actual string, strict, requireValues bool) {
	for _, k := range r.Missing {
		fmt.Fprintf(w, "::error file=%s::Missing key %q (declared in %s)\n", actual, k, template)
	}
	if requireValues {
		for _, k := range r.Empty {
			fmt.Fprintf(w, "::error file=%s::Required key %q is empty\n", actual, k)
		}
	}
	if strict {
		for _, k := range r.Extra {
			fmt.Fprintf(w, "::warning file=%s::Extra key %q not declared in %s\n", actual, k, template)
		}
	}
	for _, u := range r.Undeclared {
		for _, site := range u.CallSites {
			file, line := splitCallSite(site)
			if line == "" {
				fmt.Fprintf(w, "::error file=%s::Key %q used in code but missing from %s\n", file, u.Key, template)
			} else {
				fmt.Fprintf(w, "::error file=%s,line=%s::Key %q used in code but missing from %s\n", file, line, u.Key, template)
			}
		}
	}
}

func splitCallSite(s string) (string, string) {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}
