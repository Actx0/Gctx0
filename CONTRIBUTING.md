# Contributing to Gctx0

Thanks for your interest in contributing to the Actx0 Go client.

## Development setup

```bash
git clone https://github.com/Actx0/Gctx0.git
cd Gctx0
go test ./...
go build ./examples/...
```

Tests start a local mock API server automatically. To run against your own local Actx0 server:

```bash
GCTX0_BASE_URL=http://127.0.0.1:8000 go test ./...
```

## Pull requests

1. Open an issue first for larger changes so we can align on scope.
2. Keep changes focused and consistent with existing package style.
3. Add or update tests for behavior you change.
4. Run `go test ./...` before opening the PR.
5. Use a clear, lowercase conventional commit subject when possible
   (for example `fix: handle empty prompt config`).

## Code style

- Prefer small, readable helpers over deep abstractions.
- Keep API types next to the resource client they belong to.
- Match existing naming and blank-line spacing in nearby files.
- Do not commit secrets, access keys, or local env files.

## Reporting bugs

Use the bug report issue template and include:

- Go version (`go version`)
- Gctx0 version / commit
- Minimal reproduction steps
- Expected vs actual behavior

## Security issues

Do not open a public issue for security problems. See [SECURITY.md](SECURITY.md).

## Code of conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). By participating, you are expected to uphold it.
