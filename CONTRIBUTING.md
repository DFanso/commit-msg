# Contributing to commit-msg

Thank you for your interest in contributing to `commit-msg`! We welcome contributions from developers of all skill levels. 🎉

## 🎃 Hacktoberfest

This project is participating in [Hacktoberfest](https://hacktoberfest.com)! We welcome quality contributions throughout October and beyond.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Issue Labels](#issue-labels)

## Code of Conduct

This project adheres to the Hacktoberfest values:
- **Everyone is welcome** - We value diversity and inclusivity
- **Quantity is fun, quality is key** - We prioritize meaningful contributions
- **Short-term action, long-term impact** - Your contributions help build the future

Please be respectful and constructive in all interactions.

## How Can I Contribute?

### 🐛 Reporting Bugs

Before creating a bug report:
- Check the [existing issues](https://github.com/dfanso/commit-msg/issues) to avoid duplicates
- Collect information about the bug:
  - OS and version
  - Go version
  - Steps to reproduce
  - Expected vs actual behavior

### 💡 Suggesting Enhancements

Enhancement suggestions are welcome! Please:
- Use a clear and descriptive title
- Provide a detailed description of the proposed feature
- Explain why this enhancement would be useful

### 🔧 Good First Issues

Look for issues labeled `good first issue` or `help wanted` - these are great for newcomers!

### 📝 Documentation

Improving documentation is always appreciated:
- Fix typos or unclear instructions
- Add examples
- Improve README clarity
- Translate documentation

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/commit-msg.git
   cd commit-msg
   ```
3. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

### Prerequisites

- Go 1.23.4 or higher
- Git
- API key for either:
  - Google Gemini (`GEMINI_API_KEY`)
  - Grok (`GROK_API_KEY`)

### Environment Setup

1. Set up your environment variables:
   ```bash
   export COMMIT_LLM=gemini  # or "grok"
   export GEMINI_API_KEY=your-api-key-here
   # OR
   export GROK_API_KEY=your-api-key-here
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Run the application:
   ```bash
   go run src/main.go .
   ```

4. Build the executable:
   ```bash
   go build -o commit.exe src/main.go
   ```

### Testing Your Changes

Before submitting a PR:
1. Test the application in a Git repository
2. Verify both LLM providers work (if applicable)
3. Check for any errors or warnings
4. Test on your target platform

## Pull Request Process

1. **Update documentation** if needed (README.md, code comments)
2. **Follow the coding standards** (see below)
3. **Write clear commit messages** (ironic for this project!)
4. **Fill out the PR template** completely
5. **Link related issues** using keywords like "Fixes #123"
6. **Wait for review** - maintainers will review within 1-2 days

### PR Title Format

Use conventional commit format:
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Adding tests
- `chore:` - Maintenance tasks

Example: `feat: add support for Claude API`

## Coding Standards

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` to format your code
- Add comments for exported functions
- Keep functions small and focused
- Handle errors appropriately

### Code Example

```go
// GenerateMessage creates a commit message from git changes
func GenerateMessage(changes string, apiKey string) (string, error) {
    if changes == "" {
        return "", fmt.Errorf("no changes provided")
    }
    
    // Implementation here
    return message, nil
}
```

## Issue Labels

- `good first issue` - Good for newcomers
- `help wanted` - Extra attention needed
- `bug` - Something isn't working
- `enhancement` - New feature request
- `documentation` - Documentation improvements
- `hacktoberfest` - Eligible for Hacktoberfest
- `hacktoberfest-accepted` - Approved for Hacktoberfest

## Questions?

Feel free to:
- Open an issue with the `question` label
- Reach out to the maintainers
- Check existing issues and PRs for similar questions

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

---

Thank you for contributing to commit-msg! 🚀

