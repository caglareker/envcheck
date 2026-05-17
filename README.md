# envcheck

[![CI](https://github.com/caglareker/envcheck/actions/workflows/ci.yml/badge.svg)](https://github.com/caglareker/envcheck/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/caglareker/envcheck?sort=semver)](https://github.com/caglareker/envcheck/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/caglareker/envcheck.svg)](https://pkg.go.dev/github.com/caglareker/envcheck)
[![Go Report Card](https://goreportcard.com/badge/github.com/caglareker/envcheck)](https://goreportcard.com/report/github.com/caglareker/envcheck)
[![codecov](https://codecov.io/gh/caglareker/envcheck/branch/main/graph/badge.svg)](https://codecov.io/gh/caglareker/envcheck)
[![License: MIT](https://img.shields.io/github/license/caglareker/envcheck)](LICENSE)

Tiny Go CLI that catches missing or empty `.env` keys **before** they break your deploy. Compares `.env` against `.env.example`, validates required values, scans source code (Go, JS/TS, Python, Ruby, Rust, PHP) for env vars referenced in code but missing from your template, and emits GitHub Actions annotations on failure. Drop it into CI or a [pre-commit hook](#pre-commit-hook) — one static binary, no dependencies.

## Install

**With Go:**

```
go install github.com/caglareker/envcheck@latest
```

**Pre-built binary:** download the archive for your platform from
[Releases](https://github.com/caglareker/envcheck/releases), extract, and put
the `envcheck` binary on your `$PATH`.

Check the version:

```
envcheck --version
```

## Usage

```
envcheck --template .env.example --actual .env
```

### Flags

| Flag               | Default        | Purpose                                                                       |
|--------------------|----------------|-------------------------------------------------------------------------------|
| `--template`       | `.env.example` | Template file listing required keys                                           |
| `--actual`         | `.env`         | Env file to check                                                             |
| `--ci`             | `false`        | Exit non-zero when any problem is detected                                    |
| `--strict`         | `false`        | Also report keys present in `--actual` but not in `--template`                |
| `--require-values` | `false`        | Fail when a required key is present but empty in `--actual` (e.g. `API_KEY=`) |
| `--scan`           | _(off)_        | Scan a source directory for env-var usage and flag keys missing from template |
| `--format`         | `text`         | Output format: `text` (human readable) or `github` (Actions annotations)      |

### Exit codes

| Code | Meaning                                                                       |
|------|-------------------------------------------------------------------------------|
| `0`  | Success — or problems found but `--ci` is off                                 |
| `1`  | `--ci` and at least one of: missing keys, empty required values, extra keys, undeclared scan hits |
| `2`  | Could not read one of the env files (e.g. file not found, permission denied)  |

### Parser notes

- Lines starting with `#` are treated as comments and ignored.
- Inline comments after a value (`PORT=5432 # the port`) are stripped.
- `export FOO=bar` is supported.
- Quoted values (`KEY="value with spaces"`, `KEY='single'`) are unwrapped.
- `KEY=""` is considered empty for `--require-values`.

### Examples

Fail CI when keys are missing:

```
envcheck --ci
```

Also flag stale keys in `.env` that are no longer in the template:

```
envcheck --strict --ci
```

Fail when a required key was added but left blank (the classic "I forgot to set it in CI" bug):

```
envcheck --require-values --ci
```

Emit GitHub Actions annotations so missing keys show up inline on PRs:

```
envcheck --ci --format=github
```

Catch env vars used in code but forgotten in `.env.example`:

```
envcheck --scan ./src --ci
```

### Scan mode

`--scan <dir>` walks the given directory and looks for env-var references in
source files. Anything used in code but not declared in the template is
reported as **undeclared**, with the file + line number of each call site.

Supported languages:

| Extension              | Detected patterns                                              |
|------------------------|----------------------------------------------------------------|
| `.go`                  | `os.Getenv("X")`, `os.LookupEnv("X")`                          |
| `.js` `.jsx` `.ts` `.tsx` `.mjs` `.cjs` | `process.env.X`, `process.env["X"]`, `import.meta.env.X`, `Deno.env.get("X")` |
| `.py`                  | `os.environ["X"]`, `os.environ.get("X")`, `os.getenv("X")`     |
| `.rb`                  | `ENV["X"]`, `ENV.fetch("X")`                                   |
| `.rs`                  | `env::var("X")`, `env::var_os("X")`                            |
| `.php`                 | `getenv("X")`, `$_ENV["X"]`                                    |

Skipped directories: `.git`, `node_modules`, `vendor`, `target`, `dist`, `build`, `.next`, `.nuxt`, `__pycache__`, `.venv`, `venv`, `.tox`.

## Pre-commit hook

envcheck ships a [pre-commit](https://pre-commit.com) hook definition. Add it
to your `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/caglareker/envcheck
    rev: v0.1.0
    hooks:
      - id: envcheck
```

The hook runs `envcheck --ci` whenever a `.env*` file changes, so commits that
introduce or break env declarations fail locally instead of in CI.

Enable additional checks via `args`:

```yaml
- id: envcheck
  args: [--ci, --strict, --require-values]
```

Scan source code for undeclared env vars:

```yaml
- id: envcheck
  args: [--ci, --scan, ./src]
```

## Why

Got tired of deploys failing at runtime because someone forgot to add a new env var to the pipeline. Wanted something simpler than a shell script.

## License

MIT
