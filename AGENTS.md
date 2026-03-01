# AGENTS.md - gookit/validate

This document provides guidelines for agents working on this codebase.

## Project Overview

`gookit/validate` is a Go data validation library supporting Maps, Structs, and HTTP Requests with 70+ built-in validators.

## Build, Lint, and Test Commands

### Build
```bash
go build ./...                    # Build all packages (~20s)
go vet ./...                      # Validate code (<1s)
go fmt ./...                      # Format code (<1s)
```

### Test
```bash
go test ./...                     # Run all tests (~5s)
go test -v ./...                  # Verbose tests
go test -coverprofile="profile.cov" ./...  # With coverage
go test -bench=. -benchmem .      # Run benchmarks (~3s)
```

### Run a Single Test
```bash
go test -v -run TestName ./...   # Run specific test by name
go test -v -run TestValidators   # Example: run validator tests
```

### Linting
```bash
golangci-lint run                # Full linting (uses .golangci.yml)
go vet ./...                     # Fallback if golangci-lint fails
```

## Code Style Guidelines

### Formatting
- Use `go fmt ./...` for formatting
- Use `goimports` for import organization (standard library first, then external)
- Configure editor to strip unused imports and variables

### Naming Conventions
- **Variables/functions**: `camelCase` (e.g., `newValidation`, `validators`)
- **Exported types/functions**: `PascalCase` (e.g., `Validation`, `Struct()`, `Map()`)
- **Constants**: `PascalCase` or `CamelCase` (e.g., `Email`, `UUID`)
- **Private fields**: `camelCase` (e.g., `data`, `fieldNames`)
- **Interfaces**: `PascalCase` ending in `er` where appropriate (e.g., `DataFace`, `ValidatorFace`)

### Type Declarations
```go
// Type aliases for common maps
type M map[string]any
type MS map[string]string
type SValues map[string][]string
```

### Import Organization
```go
import (
    "bytes"
    "fmt"
    "net/http"
    "reflect"
    "regexp"
    "strings"

    "github.com/gookit/goutil/reflects"
)
```
- Group: stdlib, then external packages
- No blank lines between groups (as seen in codebase)

### Error Handling
- Use `errors.New()` and `fmt.Errorf()` for creating errors
- Return errors explicitly rather than using sentinel error variables
- Use `err != nil` checks for error handling
- Example pattern:
```go
func FromJSONBytes(bs []byte) (*MapData, error) {
    mp := map[string]any{}
    if err := Unmarshal(bs, &mp); err != nil {
        return nil, err
    }
    return &MapData{Map: mp}, nil
}
```

### Validation Patterns
When adding new validators:
1. Follow existing validator function signatures in `validators.go`
2. Use descriptive error messages with `{field}` placeholder
3. Register validators in appropriate maps
4. Add tests to corresponding `*_test.go` file
5. Test both success and failure scenarios

### Struct Tags
```go
type User struct {
    Name  string `validate:"required|min_len:2" label:"User Name"`
    Email string `validate:"email" label:"Email Address"`
    Age   int    `validate:"required|int|min:1|max:99"`
}
```
- `validate`: Validation rules (pipe-separated)
- `label`: Display name for error messages
- `filter`: Filtering/preprocessing rules
- `message`: Custom error message
- `default`: Default value

### Test Conventions
- Test files: `*_test.go` in same package
- Table-driven tests for multiple test cases:
```go
func TestValidatorName(t *testing.T) {
    tests := []struct {
        name  string
        input any
        want  bool
    }{
        {"valid email", "test@example.com", true},
        {"invalid email", "invalid", false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```
- Use `github.com/gookit/goutil/testutil/assert` for assertions

### Performance Considerations
- Library is optimized for sub-nanosecond validation
- Pre-compile regex patterns at package level
- Reuse maps/slices when possible
- Run benchmarks when modifying core logic

## Key Files

| File | Purpose |
|------|---------|
| `validate.go` | Main entry points (New, Struct, Map, Request) |
| `validation.go` | Core Validation struct and logic |
| `validators.go` | 70+ built-in validators |
| `data_source.go` | Data source abstractions |
| `filtering.go` | Data filtering/sanitization |
| `messages.go` | Error messages & i18n |
| `rule.go` | Rule parsing |
| `validating.go` | Validation execution |
| `util.go` | Utility functions |
| `value.go` | Value type conversion |

## Common Tasks

### Adding a New Validator
1. Add function in `validators.go`
2. Register in appropriate validator map
3. Add documentation in `docs/validators.md`
4. Add test cases
5. Verify with `go test -v -run YourValidator`

### Testing Validation Scenarios
```go
// Map validation
v := validate.Map(map[string]any{"name": "john"})
v.StringRule("name", "required|min_len:2")

// Struct validation
type User struct {
    Name string `validate:"required|min_len:2"`
}
v := validate.Struct(&User{Name: "john"})

ok := v.Validate()
```

## CI Commands (from .github/workflows/go.yml)
- Tests run on Go 1.19, 1.21-1.25
- Coverage reported to coveralls
- golangci-lint v2.x for linting

## Dependencies
- `github.com/gookit/filter` - Data filtering
- `github.com/gookit/goutil` - Utility functions
- `golang.org/x/sync` - Synchronization primitives
