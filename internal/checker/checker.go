package checker

import (
	"bufio"
	"os"
	"strings"
)

type Result struct {
	Missing []string
	Extra   []string
}

func Check(templatePath, actualPath string) (*Result, error) {
	required, err := readKeys(templatePath)
	if err != nil {
		return nil, err
	}
	actual, err := readKeys(actualPath)
	if err != nil {
		return nil, err
	}

	r := &Result{}
	seen := toSet(actual)
	for _, k := range required {
		if _, ok := seen[k]; !ok {
			r.Missing = append(r.Missing, k)
		}
	}
	required_set := toSet(required)
	for _, k := range actual {
		if _, ok := required_set[k]; !ok {
			r.Extra = append(r.Extra, k)
		}
	}
	return r, nil
}

func readKeys(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var keys []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		keys = append(keys, strings.TrimSpace(line[:idx]))
	}
	return keys, sc.Err()
}

func toSet(xs []string) map[string]struct{} {
	m := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		m[x] = struct{}{}
	}
	return m
}
