# Contributing to witr

Contributions are welcome and appreciated ❤️

## Issues:

- Check existing issues before opening a new one.
- Keep descriptions clear and concise.
- Steps to reproduce help a lot.

## Pull Requests:

- Fork the repo and create a feature branch.
- Write clear, concise commit messages.
- Keep PRs focused (one fix or feature per PR).
- Open PRs against the `staging` branch, not `main`.
- Update the README only if the change affects users.

## Building from source

When you need to verify a change locally, compile the CLI with Go 1.25+
so that the embedded version data stays accurate:

```bash
git clone https://github.com/SanCognition/witr.git
cd witr
go build -ldflags "-X main.version=v0.0.0-dev -X main.commit=$(git rev-parse --short HEAD) -X 'main.buildDate=$(date +%Y-%m-%d)'" -o witr ./cmd/witr
./witr --help  # quick smoke test
```

- The `-ldflags` block injects commit/date metadata for `witr --version`.
- The resulting `witr` binary lands in the repo root.

## Code Style

- Follow the existing code style and structure.
- Use `gofmt` before submitting Go code.

## Communication

- Be respectful and constructive.
- Follow the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/)

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (see LICENSE).
