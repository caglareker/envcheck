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

Exit code is 0 by default, even when keys are missing. Use `--ci` to make it fail:

```
envcheck --ci
```

## Why

Got tired of deploys failing at runtime because someone forgot to add a new env var to the pipeline. Wanted something simpler than a shell script.

## License

MIT
