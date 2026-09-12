package checker

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/caglareker/envcheck/internal/scanner"
)

type Entry struct {
	Key   string
	Value string
}

type Result struct {
	Missing    []string
	Extra      []string
	Empty      []string
	Undeclared []UndeclaredKey
	// Ignored lists keys that matched an --ignore pattern and were therefore
	// suppressed from one of the categories above.
	Ignored []string
}

// UndeclaredKey is a key referenced in source code but missing from the template.
type UndeclaredKey struct {
	Key       string
	CallSites []string
}

type Options struct {
	RequireValues bool
	ScanPath      string
	// Ignore holds path.Match glob patterns; matching keys are never reported.
	Ignore []string
}

func Check(templatePath, actualPath string, opts Options) (*Result, error) {
	ignore, err := newIgnoreMatcher(opts.Ignore)
	if err != nil {
		return nil, err
	}

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

	if opts.ScanPath != "" {
		sr, err := scanner.Scan(opts.ScanPath)
		if err != nil {
			return nil, err
		}
		var keys []string
		for k := range sr.UsedKeys {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if _, ok := requiredMap[k]; ok {
				continue
			}
			r.Undeclared = append(r.Undeclared, UndeclaredKey{Key: k, CallSites: sr.UsedKeys[k]})
		}
	}

	ignore.apply(r)

	return r, nil
}

// ignoreMatcher filters keys against a set of path.Match glob patterns.
type ignoreMatcher struct {
	patterns []string
}

func newIgnoreMatcher(patterns []string) (*ignoreMatcher, error) {
	for _, p := range patterns {
		// path.Match reports a malformed pattern regardless of the subject,
		// so an empty subject is enough to validate up front.
		if _, err := path.Match(p, ""); err != nil {
			return nil, fmt.Errorf("invalid --ignore pattern %q: %w", p, err)
		}
	}
	return &ignoreMatcher{patterns: patterns}, nil
}

func (m *ignoreMatcher) match(key string) bool {
	for _, p := range m.patterns {
		if ok, _ := path.Match(p, key); ok {
			return true
		}
	}
	return false
}

// apply strips ignored keys from every reported category and records which
// keys were actually suppressed, so a key that was never going to be reported
// is not counted.
func (m *ignoreMatcher) apply(r *Result) {
	if len(m.patterns) == 0 {
		return
	}

	suppressed := make(map[string]struct{})
	filter := func(keys []string) []string {
		var kept []string
		for _, k := range keys {
			if m.match(k) {
				suppressed[k] = struct{}{}
				continue
			}
			kept = append(kept, k)
		}
		return kept
	}
	r.Missing = filter(r.Missing)
	r.Extra = filter(r.Extra)
	r.Empty = filter(r.Empty)

	var undeclared []UndeclaredKey
	for _, u := range r.Undeclared {
		if m.match(u.Key) {
			suppressed[u.Key] = struct{}{}
			continue
		}
		undeclared = append(undeclared, u)
	}
	r.Undeclared = undeclared

	for k := range suppressed {
		r.Ignored = append(r.Ignored, k)
	}
	sort.Strings(r.Ignored)
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
