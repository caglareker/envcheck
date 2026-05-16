package scanner

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Result struct {
	UsedKeys map[string][]string // key -> sorted list of "file:line" call sites
}

var ignoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"target":       true,
	"dist":         true,
	"build":        true,
	".next":        true,
	".nuxt":        true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	".tox":         true,
}

// patterns is the map of file extension -> regexes that capture an env key
// in capture group 1. Keys must match keyPattern after extraction.
var patterns = map[string][]*regexp.Regexp{
	".go": {
		regexp.MustCompile(`os\.(?:Getenv|LookupEnv)\(\s*"([A-Za-z_][A-Za-z0-9_]*)"`),
	},
	".js":  jsPatterns(),
	".jsx": jsPatterns(),
	".ts":  jsPatterns(),
	".tsx": jsPatterns(),
	".mjs": jsPatterns(),
	".cjs": jsPatterns(),
	".py": {
		regexp.MustCompile(`os\.environ(?:\.get)?\(\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]`),
		regexp.MustCompile(`os\.environ\[\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]\s*\]`),
		regexp.MustCompile(`os\.getenv\(\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]`),
	},
	".rb": {
		regexp.MustCompile(`ENV(?:\.fetch)?\(\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]`),
		regexp.MustCompile(`ENV\[\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]\s*\]`),
	},
	".rs": {
		regexp.MustCompile(`env::var(?:_os)?\(\s*"([A-Za-z_][A-Za-z0-9_]*)"`),
	},
	".php": {
		regexp.MustCompile(`getenv\(\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]\s*\)`),
		regexp.MustCompile(`\$_ENV\[\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]\s*\]`),
	},
}

func jsPatterns() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile(`process\.env\.([A-Za-z_][A-Za-z0-9_]*)`),
		regexp.MustCompile(`process\.env\[\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]\s*\]`),
		regexp.MustCompile(`import\.meta\.env\.([A-Za-z_][A-Za-z0-9_]*)`),
		regexp.MustCompile(`Deno\.env\.get\(\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]`),
	}
}

func Scan(root string) (*Result, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan path is not a directory: %s", root)
	}

	used := make(map[string]map[string]struct{})

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if path != root && ignoredDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		regexes, ok := patterns[ext]
		if !ok {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		for lineIdx, line := range strings.Split(string(data), "\n") {
			for _, re := range regexes {
				for _, m := range re.FindAllStringSubmatch(line, -1) {
					if len(m) < 2 {
						continue
					}
					key := m[1]
					loc := fmt.Sprintf("%s:%d", rel, lineIdx+1)
					if _, ok := used[key]; !ok {
						used[key] = make(map[string]struct{})
					}
					used[key][loc] = struct{}{}
				}
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	out := make(map[string][]string, len(used))
	for k, locs := range used {
		list := make([]string, 0, len(locs))
		for l := range locs {
			list = append(list, l)
		}
		sort.Strings(list)
		out[k] = list
	}
	return &Result{UsedKeys: out}, nil
}
