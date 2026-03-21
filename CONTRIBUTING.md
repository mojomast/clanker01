# Contributing to SWARM

We welcome contributions to SWARM! This document provides guidelines for contributing to the project.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Style](#code-style)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Skill Development](#skill-development)

## Getting Started

### Prerequisites

- Go 1.22 or higher
- Git
- Basic understanding of:
  - Go programming language
  - Model Context Protocol (MCP)
  - Multi-agent AI systems
  - Terminal User Interfaces (optional)

### Setup Development Environment

```bash
# Clone the repository
git clone https://github.com/mojomast/clanker01.git
cd clanker01

# Install dependencies
go mod download

# Run tests to verify setup
go test ./...

# Build the binary
go build -o swarm ./cmd/swarm
```

### Recommended Tools

- **IDE**: VS Code with Go extension
- **Linter**: `golangci-lint`
- **Formatter**: `go fmt` (standard)
- **Testing**: `go test` with coverage

## Development Workflow

### Branch Strategy

We use a simplified Git workflow:

- `main` - Production code, always deployable
- `feature/*` - Feature branches
- `bugfix/*` - Bug fix branches
- `hotfix/*` - Urgent production fixes

### Making Changes

1. **Create a branch** for your feature/bugfix:
   ```bash
   git checkout -b feature/my-new-feature
   ```

2. **Make your changes** following code style guidelines

3. **Test thoroughly**:
   ```bash
   go test ./...
   go test -cover ./...
   ```

4. **Commit with clear messages**:
   ```bash
   git add .
   git commit -m "Add support for custom skill runtime"
   ```

5. **Push to your fork**:
   ```bash
   git push origin feature/my-new-feature
   ```

6. **Create Pull Request** (see PR Process below)

## Code Style

### Go Guidelines

- Use `gofmt` for formatting
- Use `golint` for linting
- Exported functions must have documentation comments
- Use standard library where possible
- Avoid premature optimization
- Handle errors explicitly (don't ignore with `_`)

### Documentation

- Package comments explain what the package does
- Exported functions have `// FunctionName` comments
- Complex logic has inline comments
- README.md updated for user-facing features

### Example

```go
// Package agent provides the core agent runtime for SWARM.
// It manages agent lifecycle, state machines, and task execution.
package agent

// Execute runs the agent with the given task and returns the result.
// It handles LLM calls, tool execution, and error recovery.
func (a *Agent) Execute(ctx context.Context, task *Task) (*Result, error) {
    // Implementation...
}
```

## Testing

### Test Structure

Follow the standard Go test structure:

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {"simple case", input, output, false},
        {"error case", badInput, nil, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionUnderTest(tt.input)

            if (err != nil) != tt.wantErr {
                t.Errorf("error mismatch: got %v, want %v", err, tt.wantErr)
            }

            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("output mismatch: got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test Coverage

Aim for **70%+ code coverage** on new code:

```bash
# Run with coverage
go test -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.html coverage.out

# Check specific module coverage
go test -cover ./internal/core/agent
```

### Types of Tests

1. **Unit Tests**: Test individual functions in isolation
2. **Integration Tests**: Test module interactions
3. **End-to-End Tests**: Test complete workflows
4. **Performance Tests**: Benchmark critical paths
5. **Fuzz Tests**: Fuzz-parse untrusted inputs

## Pull Request Process

### Before Submitting

- [ ] All tests pass
- [ ] Code is formatted (`go fmt ./...`)
- [ ] Linting passes (`golangci-lint run`)
- [ ] Coverage is adequate (70%+)
- [ ] Documentation is updated
- [ ] Commit messages are clear

### PR Description Template

```markdown
## Description
Brief description of what this PR does.

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation
- [ ] Performance improvement

## Testing
Describe how you tested this change:
- Unit tests pass
- Manual testing steps
- Screenshots (if applicable)

## Checklist
- [ ] Tests pass
- [ ] Code formatted
- [ ] Linting passes
- [ ] Documentation updated
- [ ] No breaking changes (or documented)
```

### Review Process

1. **Automated Checks**: CI runs tests, linting, and formatting
2. **Code Review**: Maintainers review code and provide feedback
3. **Address Feedback**: Make requested changes
4. **Approval**: PR approved after all requirements met
5. **Merge**: Squashed and merged to main branch

## Skill Development

### Creating a New Skill

Skills are modular, hot-loadable plugins. See [README.md](README.md#developing-skills) for details.

### Skill Guidelines

- **Manifest Required**: Every skill must have `skill.yaml`
- **Security Profile**: Declare appropriate permissions (restricted/standard/elevated)
- **Error Handling**: Graceful errors, proper logging
- **Testing**: Unit tests for all tool functions
- **Documentation**: Clear README with usage examples

### Skill Structure

```
my-skill/
├── skill.yaml          # Manifest file
├── skill.go           # Go implementation
├── skill_test.go      # Tests
└── README.md          # Skill documentation
```

### Contributing Built-in Skills

We welcome contributions to built-in skills:

1. Fork the repository
2. Create skill in `skills/builtin/your-skill/`
3. Implement required interfaces
4. Add comprehensive tests
5. Submit PR with skill description

## Reporting Issues

### Bug Reports

Include the following information:

```markdown
**Description**: Clear description of the bug

**Steps to Reproduce**:
1. Step one
2. Step two
3. Step three

**Expected Behavior**: What should happen

**Actual Behavior**: What actually happens

**Environment**:
- SWARM version: `swarm --version`
- OS: e.g., Linux, macOS
- Go version: `go version`
- Config: Relevant config section (redact secrets)

**Logs**: Error messages or relevant log output
```

### Feature Requests

Describe the feature:

```markdown
**Title**: Clear, concise title

**Description**: Detailed description of the feature

**Use Case**: Why is this feature needed?

**Proposed Solution**: How should it work?

**Alternatives Considered**: What other approaches were considered?
```

## Community Guidelines

### Code of Conduct

- Be respectful and inclusive
- Welcome newcomers and help them learn
- Focus on what is best for the project
- Constructive feedback only

### Communication Channels

- **GitHub Issues**: For bugs, feature requests, questions
- **Discord**: For real-time discussion and community support
- **GitHub Discussions**: For general topics and Q&A

## Getting Help

If you need help contributing:

1. Check existing issues and PRs
2. Read this document thoroughly
3. Ask in GitHub Discussions or Discord
4. Start small and iterate

## Recognition

Contributors are recognized in:
- CONTRIBUTORS.md file
- GitHub release notes
- README.md acknowledgments

Thank you for contributing to SWARM! 🚀
