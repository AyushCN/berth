# Contributing to Berth

Thank you for your interest in contributing to Berth! This guide will help you get started.

---

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md). Please report unacceptable behavior to conduct@berth.dev.

---

## Ways to Contribute

- **Bug Reports** - File issues with reproduction steps
- **Feature Requests** - Propose new functionality
- **Documentation** - Improve docs, fix typos, add examples
- **Code Contributions** - Fix bugs, add features, improve tests
- **Code Reviews** - Review PRs, suggest improvements
- **Testing** - Write tests, find edge cases
- **Community** - Help others on Discord/GitHub Discussions

---

## Getting Started

### Development Setup
```bash
# 1. Fork the repo
# 2. Clone your fork
git clone https://github.com/YOUR_USERNAME/berth.git
cd berth

# 3. Add upstream remote
git remote add upstream https://github.com/berth/berth.git

# 4. Setup development environment
# See DEVELOPMENT.md for detailed instructions

# 5. Verify setup
make test
```

### First Contribution
Look for issues tagged `good first issue` or `help wanted`.

---

## Development Workflow

### Branch Naming
```
feat/<short-description>     # New feature
fix/<short-description>      # Bug fix
docs/<short-description>     # Documentation
refactor/<short-description> # Code refactoring
test/<short-description>     # Test additions
chore/<short-description>    # Maintenance
```

### Commit Messages
Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short description>

[optional body]

[optional footer]
```

**Types:**
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation
- `style` - Formatting, no logic change
- `refactor` - Code restructuring
- `test` - Test additions/changes
- `chore` - Maintenance, deps
- `perf` - Performance improvement

**Examples:**
```
feat(analyzer): add Rust Axum framework detection
fix(worker): handle container cleanup on job failure
docs(api): add prediction endpoints to API.md
refactor(usecase): extract build strategy interface
test(analyzer): add Go Gin framework tests
```

---

## Pull Request Process

### Before Submitting
- [ ] Run `make test` - All tests pass
- [ ] Run `make lint` - No linting errors
- [ ] Update documentation if needed
- [ ] Add tests for new functionality
- [ ] Update CHANGELOG.md (Unreleased section)
- [ ] Squash commits if needed

### PR Template
```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update
- [ ] Refactoring

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing done

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex logic
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
```

### Review Process
1. **Automated Checks** - CI runs tests, linting, type-checking
2. **Code Review** - At least 1 maintainer approval required
3. **Address Feedback** - Respond to comments, push updates
4. **Merge** - Maintainer merges after approval

---

## Code Standards

### Go Style
```go
// Use gofmt/goimports
go fmt ./...
goimports -w .

// Linting
golangci-lint run ./...

// Error handling
if err != nil {
    return fmt.Errorf("context: %w", err)
}

// Context usage
func DoSomething(ctx context.Context) error {
    // Check ctx.Done() in long operations
}
```

### Frontend Style
```bash
# Format
npm run format

# Lint
npm run lint

# Type check
npm run type-check
```

### Naming Conventions
| Element | Convention | Example |
|---------|------------|---------|
| Go packages | lowercase, single word | `analyzer`, `usecase` |
| Go types | PascalCase | `RuntimeProfile` |
| Go functions | PascalCase | `AnalyzeProject` |
| Go variables | camelCase | `runtimeProfile` |
| Go constants | UPPER_SNAKE | `MaxRetries` |
| Go interfaces | PascalCase + er | `BuildStrategy` |
| TS types | PascalCase | `RuntimeProfile` |
| TS functions | camelCase | `analyzeProject` |
| TS variables | camelCase | `runtimeProfile` |
| CSS classes | kebab-case | `.file-tree-node` |
| Git branches | kebab-case | `feat/rust-axum-support` |

---

## Testing Requirements

### Unit Tests
- Test public APIs, not private functions
- Use table-driven tests for multiple cases
- Mock external dependencies
- Target > 85% coverage

### Integration Tests
- Test component interactions
- Use testcontainers for external services
- Clean up resources in `defer`

### Frontend Tests
- Test user interactions, not implementation
- Use React Testing Library
- Test accessibility (ARIA, keyboard)

---

## Documentation

### When to Update
- New features → API.md, FRONTEND.md, BACKEND.md
- Breaking changes → CHANGELOG.md, UPGRADE.md
- New config → DEPLOYMENT.md, DEVELOPMENT.md
- Security changes → SECURITY.md

### Style
- Use clear, concise language
- Include code examples
- Keep diagrams up to date (Mermaid)
- Link related docs

---

## Issue Reporting

### Bug Report Template
```markdown
## Bug Description
Clear description of the issue

## Steps to Reproduce
1. Step one
2. Step two
3. Step three

## Expected Behavior
What should happen

## Actual Behavior
What actually happens

## Environment
- OS: [e.g., Ubuntu 22.04]
- Go version: [e.g., 1.22.0]
- Node version: [e.g., 20.10.0]
- Docker version: [e.g., 24.0.0]

## Logs/Stack Trace
```
paste relevant logs here
```

## Screenshots (if applicable)
```

---

## Feature Request Template
```markdown
## Feature Description
Clear description of the proposed feature

## Use Case
Why is this needed? Who benefits?

## Proposed Solution
How should it work? Any alternatives considered?

## Implementation Considerations
- Dependencies?
- Breaking changes?
- Migration needed?
- Performance impact?

## Additional Context
Sketches, mockups, related issues
```

---

## Code Review Guidelines

### For Authors
- Keep PRs small (< 400 lines ideal)
- Write clear commit messages
- Self-review before requesting review
- Address all CI failures
- Respond to all comments

### For Reviewers
- Be constructive and specific
- Focus on logic, not style (style is automated)
- Ask questions, don't assume
- Approve when confident
- Request changes with clear reasoning

### Review Checklist
- [ ] Correctness - Does it solve the problem?
- [ ] Tests - Adequate coverage, meaningful assertions
- [ ] Security - No vulnerabilities introduced
- [ ] Performance - No regressions
- [ ] Maintainability - Clear, readable, documented
- [ ] Consistency - Follows project patterns
- [ ] Documentation - Updated if needed

---

## Release Process

### Maintainers Only
1. Create release branch: `release/v0.x.y`
2. Update version in code
3. Update CHANGELOG.md (move Unreleased → version)
5. Run full test suite: `make test && make lint`
6. Build images: `make build`
6. Tag: `git tag v0.x.y`
7. Push: `git push origin v0.x.y`
8. Build & push Docker images
9. Create GitHub Release
10. Announce

---

## Getting Help

- **Discord**: [Berth Community](https://discord.gg/berth)
- **GitHub Discussions**: [Q&A](https://github.com/berth/berth/discussions)
- **Issues**: [Bug Reports](https://github.com/berth/berth/issues)

---

## Recognition

Contributors are recognized in:
- GitHub Contributors page
- Release notes
- Annual contributor highlights

---

## License

By contributing, you agree that your contributions will be licensed under the MIT License.