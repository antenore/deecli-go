# Contributing to DeeCLI

## Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Keep functions focused and small
- Handle errors explicitly

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

1. Fork the repository
2. Create a feature branch from master
3. Make your changes
4. Run tests: `go test ./...`
5. Submit pull request against master

## Testing

- Write tests for new functionality
- Ensure existing tests pass
- Test on multiple platforms if possible

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