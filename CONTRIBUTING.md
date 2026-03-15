# Contributing to Stepmark

Thanks for your interest in contributing! Here's how to get started.

## Getting Started

```bash
git clone https://github.com/ImVivec/stepmark.git
cd stepmark
make check   # runs fmt, vet, and tests
```

## Development

Stepmark is a **zero-dependency** library. The core package must only use the Go standard library.

### Prerequisites

- Go 1.23 or later
- (Optional) [golangci-lint](https://golangci-lint.run/welcome/install/) for `make lint`

### Make Targets

| Target | What it does |
|--------|-------------|
| `make test` | Run tests with race detector |
| `make bench` | Run benchmarks |
| `make vet` | Run `go vet` |
| `make lint` | Run golangci-lint |
| `make fmt` | Format code |
| `make cover` | Generate HTML coverage report |
| `make check` | Format + vet + test (run before submitting) |

### Running Tests

```bash
make test              # all tests with race detector
make bench             # benchmarks
make cover             # coverage report → coverage.html
```

## Submitting Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run `make check` to verify everything passes
5. Commit with a clear message
6. Push and open a Pull Request

### Commit Messages

Use concise, descriptive commit messages:

```
Add WithSampling option for probabilistic tracing
Fix race condition in entity metadata merge
```

### Pull Request Guidelines

- One logical change per PR
- Include tests for new functionality
- Update documentation if the public API changes
- Benchmarks must not regress — run `make bench` and include results for performance-sensitive changes
- All CI checks must pass

## Code Standards

- Follow standard Go conventions (`gofmt`, `go vet`)
- Exported functions must have godoc comments
- Tests use the `testing` package — no test framework dependencies
- Benchmarks go in `bench_test.go`
- Examples go in `example_test.go` (they appear on pkg.go.dev)

## Performance

Stepmark's core promise is **zero overhead when disabled**. Any change to the hot path must:

1. Maintain < 2ns / 0 allocs for disabled-path calls
2. Not introduce new allocations on the disabled path
3. Include benchmark results in the PR description

## Reporting Issues

Use [GitHub Issues](https://github.com/ImVivec/stepmark/issues). Include:

- Go version (`go version`)
- OS and architecture
- Minimal reproduction code
- Expected vs actual behavior

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
