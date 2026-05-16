package checker

import (
	"bufio"
	"os"
	"strings"
)

type Entry struct {
	Key   string
	Value string
}

type Result struct {
	Missing []string
	Extra   []string
	Empty   []string
}

type Options struct {
	RequireValues bool
}

func Check(templatePath, actualPath string, opts Options) (*Result, error) {
	required, err := readEntries(templatePath)
	if err != nil {
		return nil, err
	}
	actual, err := readEntries(actualPath)
	if err != nil {
		return nil, err
	}

	actualMap := toMap(actual)
	requiredMap := toMap(required)

	r := &Result{}
	for _, e := range required {
		if _, ok := actualMap[e.Key]; !ok {
			r.Missing = append(r.Missing, e.Key)
		}
	}
	for _, e := range actual {
		if _, ok := requiredMap[e.Key]; !ok {
			r.Extra = append(r.Extra, e.Key)
		}
	}

	if opts.RequireValues {
		for _, e := range required {
			actualVal, present := actualMap[e.Key]
			if !present {
				continue
			}
			if actualVal == "" {
				r.Empty = append(r.Empty, e.Key)
			}
		}
	}

	return r, nil
}

func readEntries(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if key == "" {
			continue
		}
		value := strings.TrimSpace(line[idx+1:])
		value = stripInlineComment(value)
		value = unquote(value)
		entries = append(entries, Entry{Key: key, Value: value})
	}
	return entries, sc.Err()
}

func stripInlineComment(s string) string {
	if strings.HasPrefix(s, `"`) || strings.HasPrefix(s, `'`) {
		return s
	}
	if i := strings.Index(s, " #"); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func unquote(s string) string {
	if len(s) < 2 {
		return s
	}
	first, last := s[0], s[len(s)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

func toMap(es []Entry) map[string]string {
	m := make(map[string]string, len(es))
	for _, e := range es {
		m[e.Key] = e.Value
	}
	return m
}
