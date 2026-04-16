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
	ci := flag.Bool("ci", false, "exit non-zero when keys are missing")
	flag.Parse()

	result, err := checker.Check(*template, *actual)
	if err != nil {
		fmt.Fprintln(os.Stderr, "envcheck:", err)
		os.Exit(2)
	}

	if len(result.Missing) == 0 {
		fmt.Println("all required keys present")
		return
	}

	fmt.Printf("missing keys in %s:\n", *actual)
	for _, k := range result.Missing {
		fmt.Printf("  - %s\n", k)
	}

	if *ci {
		os.Exit(1)
	}
}
