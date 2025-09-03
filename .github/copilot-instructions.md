# gookit/validate - Go Data Validation Library

Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

gookit/validate is a generic Go data validation and filtering library that supports validating Maps, Structs, and HTTP Request data with over 70 built-in validators and filters.

## Working Effectively

### Bootstrap and Setup
- Ensure Go 1.18+ is installed (supports Go 1.19-1.24 per CI)
- Clone repository: `git clone https://github.com/gookit/validate.git`
- Download dependencies: `go mod download` -- takes ~2 seconds
- Verify compilation: `go build ./...` -- takes ~20 seconds

### Build and Test
- Build all packages: `go build ./...` -- takes ~20 seconds, NEVER CANCEL, set timeout to 60+ seconds
- Run all tests: `go test ./...` -- takes ~5 seconds, NEVER CANCEL, set timeout to 30+ seconds  
- Run tests with coverage (CI command): `go test -coverprofile="profile.cov" ./..."` -- takes ~2 seconds
- Run benchmarks: `go test -bench=. -benchmem .` -- takes ~3 seconds
- Validate code: `go vet ./...` -- takes <1 second
- Format code: `go fmt ./...` -- takes <1 second

### Linting
- The project uses golangci-lint but has environmental issues in some setups
- If golangci-lint fails, use: `go vet ./...` as a reliable alternative
- Always run `go vet ./...` and `go fmt ./...` before committing
- CI uses golangci-lint v1.53 with specific linter configuration in `.golangci.yml`

## Validation

### Manual Testing Scenarios
Always test your changes by running validation scenarios:

1. **Basic Map Validation**:
```go
m := map[string]any{
    "name":  "john",
    "age":   25,
    "email": "john@example.com",
}
v := validate.Map(m)
v.StringRule("name", "required|min_len:2")
v.StringRule("age", "required|int|min:1|max:99") 
v.StringRule("email", "required|email")
// Should return: validation passes
```

2. **Struct Validation**:
```go
type User struct {
    Name  string `validate:"required|min_len:2" label:"User Name"`
    Email string `validate:"email" label:"Email Address"`
    Age   int    `validate:"required|int|min:1|max:99" label:"Age"`
}
user := &User{Name: "Alice", Email: "alice@example.com", Age: 30}
v := validate.Struct(user)
// Should return: validation passes
```

3. **Validation with Errors**:
```go
m := map[string]any{"name": "x", "age": 150, "email": "invalid"}
v := validate.Map(m)
v.StringRule("name", "required|min_len:2")
v.StringRule("age", "required|int|min:1|max:99")
v.StringRule("email", "required|email")
// Should return: validation fails with specific error messages
```

4. **Test HTTP Examples**:
- Run examples in `_examples/httpdemo/`: `cd _examples/httpdemo && go run main.go`
- Test HTTP validation endpoints if examples include servers

### Validation Steps for Changes
1. **Always run the full test suite** to ensure no regressions
2. **Test with both valid and invalid data** to verify error handling
3. **Check internationalization** by testing with different locales in `locales/`
4. **Verify struct tag parsing** works correctly with custom validators
5. **Test performance** with benchmarks if modifying core validation logic

## Common Tasks

### Repository Structure
```
.
├── README.md                    # Main documentation
├── go.mod                      # Go module definition
├── .golangci.yml              # Linter configuration
├── .github/workflows/go.yml   # CI/CD pipeline
├── _examples/                 # Usage examples
│   ├── httpdemo/             # HTTP validation examples
│   └── httpdemo2/            # Additional HTTP examples
├── locales/                   # Internationalization
│   ├── zhcn/                 # Chinese Simplified
│   ├── zhtw/                 # Chinese Traditional  
│   └── ruru/                 # Russian
├── docs/                      # Documentation
│   ├── validators.md         # Built-in validators reference
│   └── diff-with-go-validator.md
├── testdata/                  # Test data files
└── *.go                      # Core library files
```

### Key Source Files
- `validate.go` - Main validation entry points and public API
- `validation.go` - Core validation logic and Validation struct  
- `validators.go` - Built-in validator implementations (70+ validators)
- `data_source.go` - Data source abstractions (Map, Struct, HTTP Request)
- `filtering.go` - Data filtering and conversion logic
- `messages.go` - Error message handling and internationalization
- `rule.go` - Validation rule parsing and management
- `validating.go` - Core validation execution logic
- `util.go` - Utility functions for validation operations
- `value.go` - Value type conversions and handling

### Test Files Coverage
The project has 17 test files with 96.4% code coverage:
- `*_test.go` - Unit tests for corresponding source files
- `issues_test.go` - Regression tests for GitHub issues
- `benchmark_test.go` - Performance benchmarks

### Examples and Usage
- Check `_examples/` directory for practical usage patterns
- HTTP validation examples in `_examples/httpdemo/`
- README.md contains comprehensive usage examples
- Use examples as templates for new validation scenarios

### Development Workflow
1. **Create tests first** for new validators or functionality
2. **Run tests frequently** during development: `go test ./...`
3. **Validate with examples** by running existing example code
4. **Check formatting and vetting** before commits: `go fmt ./... && go vet ./...`
5. **Test validation scenarios manually** using the patterns from examples above
6. **Verify CI compatibility** by running the same commands as `.github/workflows/go.yml`
7. **Check specific file modifications**:
   - If modifying `validators.go`, run validation tests: `go test -run TestValidators`
   - If modifying `data_source.go`, test with different data types
   - If modifying `messages.go`, test with different locales

### Troubleshooting
- If golangci-lint fails with "depguard" errors, use `go vet ./...` instead
- All builds and tests should complete within 30 seconds on modern hardware
- If tests fail, check if they're related to your changes or pre-existing issues
- Check `issues_test.go` for known issue patterns and solutions
- Use `-v` flag with go test for verbose output: `go test -v ./...`

### Performance Considerations
- The library is highly optimized with sub-nanosecond validation for simple cases
- Benchmark results show ~2.5ns/op for field validation
- When modifying core logic, always run benchmarks to verify performance
- Use `go test -bench=. -benchmem .` to check memory allocations

## Key Features to Understand
- **Multi-source validation**: Maps, Structs, HTTP Requests
- **Filtering/Sanitization**: Data conversion before validation  
- **Scene-based validation**: Different rules for different contexts
- **Custom validators**: Add domain-specific validation logic
- **Internationalization**: Built-in support for multiple languages
- **Error handling**: Detailed error messages with field mapping
- **Tag-based configuration**: Configure validation via struct tags

Always ensure that any new validators or filters follow the existing patterns and maintain the library's performance characteristics.

## Testing Requirements
- NEVER CANCEL long-running operations - builds may take up to 30 seconds
- Set timeouts to 60+ seconds for build commands and 30+ seconds for test commands
- Always test both success and failure scenarios for validation logic
- Verify error messages are helpful and correctly formatted
- Test with different data types and edge cases
- Check memory usage for performance-sensitive changes