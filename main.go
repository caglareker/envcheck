package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/caglareker/envcheck/internal/checker"
)

func main() {
	template := flag.String("template", ".env.example", "template env file with required keys")
	actual := flag.String("actual", ".env", "actual env file to check")
	ci := flag.Bool("ci", false, "exit non-zero when problems are found")
	strict := flag.Bool("strict", false, "also report keys present in actual but not in template")
	requireValues := flag.Bool("require-values", false, "fail when a required key is present but empty in actual")
	format := flag.String("format", "text", "output format: text|github")
	flag.Parse()

	opts := checker.Options{RequireValues: *requireValues}
	result, err := checker.Check(*template, *actual, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "envcheck: could not read env files: %v\n", err)
		os.Exit(2)
	}

	switch *format {
	case "github":
		printGitHub(result, *template, *actual, *strict, *requireValues)
	default:
		printText(result, *template, *actual, *strict, *requireValues)
	}

	if *ci {
		if len(result.Missing) > 0 {
			os.Exit(1)
		}
		if *requireValues && len(result.Empty) > 0 {
			os.Exit(1)
		}
		if *strict && len(result.Extra) > 0 {
			os.Exit(1)
		}
	}
}

func printText(r *checker.Result, template, actual string, strict, requireValues bool) {
	clean := len(r.Missing) == 0 &&
		!(requireValues && len(r.Empty) > 0) &&
		!(strict && len(r.Extra) > 0)
	if clean {
		fmt.Printf("%s has all keys from %s\n", actual, template)
		return
	}

	if len(r.Missing) > 0 {
		fmt.Printf("%s is missing %d key(s) defined in %s:\n", actual, len(r.Missing), template)
		for _, k := range r.Missing {
			fmt.Printf("  - %s\n", k)
		}
	}
	if requireValues && len(r.Empty) > 0 {
		fmt.Printf("%s has %d empty value(s) for required key(s):\n", actual, len(r.Empty))
		for _, k := range r.Empty {
			fmt.Printf("  ! %s\n", k)
		}
	}
	if strict && len(r.Extra) > 0 {
		fmt.Printf("%s has %d extra key(s) not in %s:\n", actual, len(r.Extra), template)
		for _, k := range r.Extra {
			fmt.Printf("  + %s\n", k)
		}
	}
}

func printGitHub(r *checker.Result, template, actual string, strict, requireValues bool) {
	for _, k := range r.Missing {
		fmt.Printf("::error file=%s::Missing key %q (declared in %s)\n", actual, k, template)
	}
	if requireValues {
		for _, k := range r.Empty {
			fmt.Printf("::error file=%s::Required key %q is empty\n", actual, k)
		}
	}
	if strict {
		for _, k := range r.Extra {
			fmt.Printf("::warning file=%s::Extra key %q not declared in %s\n", actual, k, template)
		}
	}
}
