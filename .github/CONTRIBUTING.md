# Contributing to gopantic

Thank you for your interest in contributing to gopantic! This document provides guidelines and information for contributors.

## Getting Started

### Prerequisites

- Go 1.24 or later
- Git
- Make (for using the Makefile)

### Setting Up Your Development Environment

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/yourusername/gopantic.git
   cd gopantic
   ```

3. Set up the development environment:
   ```bash
   make init
   ```
   This will install development dependencies and set up git hooks.

4. Verify everything works:
   ```bash
   make dev
   ```

## Development Workflow

### Making Changes

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes, following the coding standards below

3. Run the development cycle frequently:
   ```bash
   make dev  # Runs deps, fmt, vet, lint, test
   ```

4. Commit your changes with a clear message:
   ```bash
   git commit -m "feat: add new validation rule for email addresses"
   ```

### Coding Standards

- **Go Code Style** - Follow standard Go formatting (`go fmt`)
- **Linting** - All code must pass `golangci-lint run`
- **Testing** - Maintain >90% test coverage for new code
- **Documentation** - Add GoDoc comments for all public APIs
- **Error Handling** - Proper error handling and meaningful error messages

### Commit Message Convention

We follow conventional commits format:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Types:
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Code style changes (formatting, etc)
- `refactor`: Code changes that neither fix bugs nor add features
- `perf`: Performance improvements
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to build process or auxiliary tools

Examples:
- `feat: add support for time.Time parsing`
- `fix: handle nil pointers in validation`
- `docs: add examples for custom validators`

## Testing

### Running Tests

```bash
make test        # Run tests with coverage
make bench       # Run benchmarks
make examples    # Test all examples
```

### Writing Tests

- Use table-driven tests where appropriate
- Test both happy path and error cases
- Include benchmarks for performance-critical code
- Add examples that demonstrate usage

Example test structure:

```go
func TestParseInto(t *testing.T) {
    tests := []struct {
        name    string
        input   []byte
        want    User
        wantErr bool
    }{
        {
            name:  "valid user",
            input: []byte(`{"id": 1, "name": "John"}`),
            want:  User{ID: 1, Name: "John"},
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseInto[User](tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseInto() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ParseInto() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Coverage Requirements

- New code should maintain >70% test coverage
- Run `make coverage` to generate and view coverage reports
- Critical paths (parsing, validation) should have 100% coverage
- Current project coverage: ~54% (contributions to improve coverage are welcome!)

## Documentation

### Code Documentation

- All public functions, types, and constants must have GoDoc comments
- Comments should explain what the code does, not how it works
- Include examples in documentation where helpful

### Examples

- Add runnable examples in the `examples/` directory
- Examples should demonstrate real-world usage
- Keep examples focused on a single feature or use case

## Pull Request Process

1. **Before submitting:**
   - Ensure all tests pass: `make test`
   - Run the full development cycle: `make dev`
   - Update documentation if needed
   - Add or update tests for your changes

2. **PR Description:**
   - Clearly describe what your PR does
   - Reference any related issues
   - Include testing instructions if applicable
   - List any breaking changes

3. **Review Process:**
   - All PRs require review from a maintainer
   - Address review feedback promptly
   - Keep PR scope focused and atomic

4. **Merging:**
   - PRs are squash-merged to maintain clean history
   - Ensure commit message follows conventional format

## Issue Reporting

### Bug Reports

Include the following information:
- Go version
- Operating system
- Minimal code example that reproduces the issue
- Expected vs actual behavior
- Error messages (if any)

### Feature Requests

- Describe the use case and problem you're trying to solve
- Provide examples of how the feature would be used
- Consider backward compatibility implications

## Architecture Guidelines

### Design Principles

- **Idiomatic Go** - APIs should feel natural in Go
- **Practical over perfect** - Focus on useful features over full parity with other libraries
- **Performance conscious** - Avoid unnecessary allocations and reflection where possible
- **Minimal dependencies** - Prefer standard library solutions

### Code Organization

```
pkg/model/           # Core library code
├── parse.go         # Main ParseInto implementation
├── validate.go      # Validation framework
├── coerce.go        # Type coercion logic
└── errors.go        # Error types and handling

examples/            # Usage examples
tests/              # Test files
```

### Error Handling

- Use structured error types that provide context
- Aggregate validation errors where appropriate
- Include field paths in validation errors
- Provide helpful error messages for users

## Community Guidelines

- Be respectful and inclusive
- Help newcomers get started
- Share knowledge and best practices
- Focus on constructive feedback

## Getting Help

- Check existing issues and documentation first
- Ask questions in GitHub issues with the "question" label
- Be specific about what you're trying to achieve

---

Thank you for contributing to gopantic!