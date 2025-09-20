# Contributing to DeeCLI

## Code Style

- Follow standard Go conventions
- Run `gofmt` or `goimports` before committing
- Keep functions focused and small
- Handle errors explicitly
- Use testify framework for all tests
- Maintain strict separation of concerns (see DEVELOPMENT.md)
- Follow established module patterns in `internal/` directory

## Commit Messages

- Use imperative mood: "Add feature" not "Added feature"
- Keep first line under 50 characters
- Reference issues when applicable: "Fix #123"
- No emojis or decorative elements

Example:
```
Fix tab completion for hidden files

- Include dot files when prefix starts with dot
- Skip hidden files by default
```

## Pull Requests

**Development Workflow:**

1. **Planning Phase**
   - Check TODO.md for current priorities
   - Review DEVELOPMENT.md for architecture patterns
   - Search codebase for existing implementations

2. **Implementation Phase**
   - Fork repository and create feature branch from master
   - Write tests FIRST using testify framework
   - Follow existing patterns and module structure
   - Implement both chat commands and CLI versions

3. **Validation Phase**
   - Run `make test-coverage` to verify coverage improvements
   - Test in real terminal environments (SSH, different terminals)
   - Ensure all tests pass: `make test`
   - Submit pull request against master

**PR Requirements:**
- Unit tests for all new functionality
- Integration tests for module interactions
- Coverage maintained or improved
- Code follows Go conventions and architecture patterns

## Testing

**Add tests for new features and bug fixes:**

```bash
make test               # All tests (required before PR)
make test-unit          # Unit tests only
make test-coverage      # Tests with HTML coverage report
make test-race          # Race condition detection
```

**Requirements:**
- **Add tests** for new features or when fixing bugs
- **Test location**: Alongside source code (`*_test.go` files)
- **Standard Go testing** - Use built-in testing package
- **Real environment testing** - Test in actual terminals and SSH connections

Keep it simple. Follow Go conventions.

## Bug Reports

Include:
- DeeCLI version
- Operating system
- Steps to reproduce
- Expected vs actual behavior
- Error messages if any

## Feature Requests

- Check existing issues first
- Describe the problem being solved
- Provide use case examples

## Questions

Open an issue with the question label.

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.