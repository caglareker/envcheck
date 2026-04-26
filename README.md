# envcheck

Small CLI to check that a `.env` file has all the keys listed in a `.env.example` template. Useful in CI when you want to catch missing env vars before deploy.

## Install

```
go install github.com/caglareker/envcheck@latest
```

## Usage

```
envcheck --template .env.example --actual .env
```

### Flags

| Flag         | Default        | Purpose                                                                       |
|--------------|----------------|-------------------------------------------------------------------------------|
| `--template` | `.env.example` | Template file listing required keys                                           |
| `--actual`   | `.env`         | Env file to check                                                             |
| `--ci`       | `false`        | Exit non-zero when keys are missing (or, with `--strict`, when extras exist)  |
| `--strict`   | `false`        | Also report keys present in `--actual` but not in `--template`                |

### Exit codes

| Code | Meaning                                                                       |
|------|-------------------------------------------------------------------------------|
| `0`  | Success — or missing/extra keys found but `--ci` is off                       |
| `1`  | `--ci` and missing keys (always), or `--ci --strict` and extra keys           |
| `2`  | Could not read one of the env files (e.g. file not found, permission denied)  |

### Examples

Fail CI when keys are missing:

```
envcheck --ci
```

Also flag stale keys in `.env` that are no longer in the template:

```
envcheck --strict --ci
```

## Why

Got tired of deploys failing at runtime because someone forgot to add a new env var to the pipeline. Wanted something simpler than a shell script.

## License

MIT
