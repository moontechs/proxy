# Linting Setup

This document describes the linting and code quality tools configured for the proxy project.

## Tools

### 1. go fmt
Standard Go formatter ensuring consistent code formatting.

```bash
make fmt
# or
go fmt ./...
```

### 2. goimports
Manages import statements and applies `go fmt`.

```bash
make fmt
# or
goimports -w .
```

**Install:**
```bash
go install golang.org/x/tools/cmd/goimports@latest
```

### 3. go vet
Official Go static analysis tool that examines Go source code and reports suspicious constructs.

```bash
make vet
# or
go vet ./...
```

**Checks for:**
- Printf format string mismatches
- Unreachable code
- Suspicious constructs
- IPv6 address formatting issues
- And more

### 4. staticcheck
Advanced static analysis tool for Go code.

```bash
make lint
# or
staticcheck ./...
```

**Install:**
```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

**Checks for:**
- Unused code
- Inefficient constructs
- Code simplifications
- Style issues
- And many more checks

### 5. golangci-lint (Optional)
Meta-linter that runs multiple linters in parallel.

```bash
make lint-golangci
```

**Install:**
```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

**Configuration:** `.golangci.yml`

Includes 30+ linters:
- errcheck
- gosimple
- govet
- ineffassign
- staticcheck
- unused
- gofmt
- goimports
- misspell
- revive
- gosec
- bodyclose
- And more...

## Quick Start

### Install Tools

```bash
make install-tools
```

This installs:
- `goimports` - Import management and formatting
- `staticcheck` - Advanced static analysis

### Run All Checks

```bash
make check
```

This runs:
1. `make fmt` - Format code
2. `make vet` - Go vet analysis
3. `make lint` - Staticcheck analysis
4. `make test` - Run tests

## Integration with Development Workflow

### Before Committing

```bash
make check
```

Ensures:
- Code is formatted
- No vet warnings
- No lint issues
- All tests pass

## Resources

- [Go vet documentation](https://pkg.go.dev/cmd/vet)
- [staticcheck documentation](https://staticcheck.io/)
- [golangci-lint documentation](https://golangci-lint.run/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
