# commit-msg

[![Hacktoberfest](https://img.shields.io/badge/Hacktoberfest-2025-orange.svg)](https://hacktoberfest.com/)
[![Go Version](https://img.shields.io/badge/Go-1.23.4-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

`commit-msg` is a command-line tool that generates commit messages using LLM (Large Language Models). It is designed to help developers create clear and concise commit messages for their Git repositories automatically by analyzing your staged changes.

## Screenshot

Below is a sample execution of `commit-msg`:

![Commit-msg Screenshot](image.png)

Before running the application, ensure you have set the system environment variables. and add commit.exe to path variables (same for linux macOS)

## 🎃 Hacktoberfest 2025

This project is participating in **Hacktoberfest 2025**! We welcome contributions from developers of all skill levels.

Looking to contribute? Check out:
- [Good First Issues](https://github.com/dfanso/commit-msg/labels/good%20first%20issue)
- [Help Wanted Issues](https://github.com/dfanso/commit-msg/labels/help%20wanted)
- [Contributing Guidelines](CONTRIBUTING.md)

## Features

✨ **AI-Powered Commit Messages** - Automatically generate meaningful commit messages  
🔄 **Multiple LLM Support** - Choose between Google Gemini or Grok  
📝 **Context-Aware** - Analyzes staged and unstaged changes  
🚀 **Easy to Use** - Simple CLI interface  
⚡ **Fast** - Quick generation of commit messages  

## Supported LLM Providers

You can use either **Google Gemini** or **Grok** as the LLM to generate commit messages:

### Environment Variables

| Variable | Values | Description |
|----------|--------|-------------|
| `COMMIT_LLM` | `gemini` or `grok` | Choose your LLM provider |
| `GEMINI_API_KEY` | Your API key | Required if using Gemini |
| `GROK_API_KEY` | Your API key | Required if using Grok |

---

## 📦 Installation

### Option 1: Download Pre-built Binary (Recommended)

1. Download the latest release from the [GitHub Releases](https://github.com/dfanso/commit-msg/releases) page
2. Extract the executable to a directory
3. Add the directory to your system PATH:

   **Windows:**
   ```cmd
   setx PATH "%PATH%;C:\path\to\commit-msg"
   ```

   **Linux/macOS:**
   ```bash
   export PATH=$PATH:/path/to/commit-msg
   echo 'export PATH=$PATH:/path/to/commit-msg' >> ~/.bashrc  # or ~/.zshrc
   ```

4. Set up environment variables:

   **Windows:**
   ```cmd
   setx COMMIT_LLM "gemini"
   setx GEMINI_API_KEY "your-api-key-here"
   ```

   **Linux/macOS:**
   ```bash
   export COMMIT_LLM=gemini
   export GEMINI_API_KEY=your-api-key-here
   # Add to ~/.bashrc or ~/.zshrc to persist
   ```

### Option 2: Build from Source

Requirements: Go 1.23.4 or higher

```bash
# Clone the repository
git clone https://github.com/dfanso/commit-msg.git
cd commit-msg

# Install dependencies
go mod download

# Build the executable
go build -o commit src/main.go

# (Optional) Install to GOPATH
go install
```

---

## 🚀 Usage

### Basic Usage

Navigate to any Git repository and run:

```bash
commit .
```

Or if running from source:

```bash
go run src/main.go .
```

### Example Workflow

```bash
# Make changes to your code
echo "console.log('Hello World')" > app.js

# Stage your changes
git add .

# Generate commit message
commit .

# Output: "feat: add hello world console log to app.js"
```

### Use Cases

- 📝 Generate commit messages for staged changes
- 🔍 Analyze both staged and unstaged changes
- 📊 Get context from recent commits
- ✅ Create conventional commit messages

---

## 🔧 Configuration

### Getting API Keys

**Google Gemini:**
1. Visit [Google AI Studio](https://makersuite.google.com/app/apikey)
2. Create a new API key
3. Set the `GEMINI_API_KEY` environment variable

**Grok (X.AI):**
1. Visit [X.AI Console](https://console.x.ai/)
2. Generate an API key
3. Set the `GROK_API_KEY` environment variable

---

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

### Quick Start for Contributors

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes
4. Commit your changes: `git commit -m 'feat: add amazing feature'`
5. Push to the branch: `git push origin feature/amazing-feature`
6. Open a Pull Request

### Areas Where We Need Help

- 🐛 Bug fixes
- ✨ New LLM provider integrations (OpenAI, Claude, etc.)
- 📚 Documentation improvements
- 🧪 Test coverage
- 🌍 Internationalization
- ⚡ Performance optimizations

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- Thanks to all [contributors](https://github.com/dfanso/commit-msg/graphs/contributors)
- Google Gemini and X.AI Grok for providing LLM APIs
- The open-source community

---

## 📞 Support

- 🐛 [Report a Bug](https://github.com/dfanso/commit-msg/issues/new?template=bug_report.md)
- 💡 [Request a Feature](https://github.com/dfanso/commit-msg/issues/new?template=feature_request.md)
- 💬 [Ask a Question](https://github.com/dfanso/commit-msg/issues)

---

Made with ❤️ for Hacktoberfest 2025




