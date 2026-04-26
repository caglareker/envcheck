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
	ci := flag.Bool("ci", false, "exit non-zero when keys are missing (or extra, in --strict mode)")
	strict := flag.Bool("strict", false, "also report keys present in actual but not in template")
	flag.Parse()

	result, err := checker.Check(*template, *actual)
	if err != nil {
		fmt.Fprintf(os.Stderr, "envcheck: could not read env files: %v\n", err)
		os.Exit(2)
	}

	if len(result.Missing) == 0 {
		fmt.Printf("%s has all keys from %s\n", *actual, *template)
	} else {
		fmt.Printf("%s is missing %d key(s) defined in %s:\n", *actual, len(result.Missing), *template)
		for _, k := range result.Missing {
			fmt.Printf("  - %s\n", k)
		}
	}

	if *strict && len(result.Extra) > 0 {
		fmt.Printf("%s has %d extra key(s) not in %s:\n", *actual, len(result.Extra), *template)
		for _, k := range result.Extra {
			fmt.Printf("  + %s\n", k)
		}
	}

	if *ci {
		if len(result.Missing) > 0 {
			os.Exit(1)
		}
		if *strict && len(result.Extra) > 0 {
			os.Exit(1)
		}
	}
}
