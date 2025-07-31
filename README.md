# CODA - AI-Powered Coding Assistant

<div align="center">

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI Status](https://github.com/common-creation/coda/workflows/CI/badge.svg)](https://github.com/common-creation/coda/actions)

An intelligent command-line coding assistant that helps you write, understand, and manage code through natural language interaction.

</div>

## Features

- 🤖 **Multi-Model Support**: Works with OpenAI GPT and Azure OpenAI models
- 💬 **Interactive Chat**: Natural language interface for coding tasks
- 🛠️ **Tool Integration**: Built-in file operations (read, write, edit, search)
- 🔒 **Security First**: Sandboxed file access with approval system
- 📝 **Context Awareness**: Understands your project structure and dependencies
- 🎨 **Rich Terminal UI**: Beautiful interface powered by Bubbletea (coming soon)
- ⚡ **Streaming Responses**: Real-time output as the AI generates responses
- 🔧 **Configurable**: Extensive configuration options via YAML/environment

## Quick Start

### Installation

#### Using Go

```bash
go install github.com/common-creation/coda@latest
```

#### From Source

```bash
git clone https://github.com/common-creation/coda.git
cd coda
make build
```

### Configuration

1. Initialize configuration:
```bash
coda config init
```

2. Set your API key:
```bash
# For OpenAI
coda config set-api-key openai

# For Azure OpenAI
coda config set-api-key azure
```

3. (Optional) Configure settings:
```bash
coda config set ai.model gpt-4
coda config set ai.temperature 0.7
```

### Basic Usage

#### Interactive Chat Mode

```bash
# Start interactive chat
coda chat

# Use a specific model
coda chat --model gpt-4

# Continue previous session
coda chat --continue
```

#### Non-Interactive Mode

```bash
# Single prompt
coda run "Explain this code: print('Hello, World!')"

# Process file
coda run --input main.go "Add comments to this Go code"

# Save output
coda run --output analysis.md "Analyze the architecture of this project"

# Pipe input
cat error.log | coda run "What's causing this error?"
```

## Commands

### `coda chat`
Start an interactive chat session with the AI assistant.

**Options:**
- `--model`: Specify AI model to use
- `--no-stream`: Disable streaming responses
- `--continue`: Continue last session
- `--no-tools`: Disable tool execution

### `coda run`
Execute a single prompt in non-interactive mode.

**Options:**
- `--input, -i`: Read input from file
- `--output, -o`: Save output to file
- `--model, -m`: Specify AI model

### `coda config`
Manage CODA configuration.

**Subcommands:**
- `show`: Display current configuration
- `set KEY VALUE`: Set a configuration value
- `get KEY`: Get a specific value
- `init`: Initialize configuration file
- `validate`: Check configuration validity
- `set-api-key`: Securely store API keys

### `coda version`
Display version information.

**Options:**
- `--verbose, -v`: Show detailed version info
- `--json`: Output as JSON

## Configuration

CODA looks for configuration in these locations (in order):
1. Command line flag: `--config`
2. Environment variable: `CODA_CONFIG`
3. `$HOME/.coda/config.yaml`
4. `./config.yaml`

### Configuration File Example

```yaml
ai:
  provider: openai  # or "azure"
  model: gpt-4
  temperature: 0.7
  max_tokens: 2048

tools:
  enabled: true
  auto_approve: false
  allowed_paths:
    - "."
  denied_paths:
    - "/etc"
    - "/sys"

session:
  max_history: 100
  max_tokens: 8000
  persistence: true

logging:
  level: info
  file: ~/.coda/coda.log
```

### Environment Variables

All configuration options can be set via environment variables:

```bash
export CODA_AI_PROVIDER=openai
export CODA_AI_MODEL=gpt-4
export CODA_AI_API_KEY=sk-...
```

## Workspace Configuration

CODA supports project-specific configuration through workspace files:

### `.coda/CODA.md` or `.claude/CLAUDE.md`

```markdown
# Project Instructions

This is a React TypeScript project using Next.js 14.

## Rules
- Always use TypeScript strict mode
- Prefer functional components with hooks
- Follow the project's ESLint configuration

## Context
- Main API endpoint: /api/v1
- Database: PostgreSQL with Prisma ORM
- Authentication: NextAuth.js
```

## Available Tools

CODA includes several built-in tools for file operations:

- **read_file**: Read contents of a file
- **write_file**: Create or overwrite a file
- **edit_file**: Modify specific parts of a file
- **list_files**: List directory contents
- **search_files**: Search for files by content or name

All tool operations require user approval by default for security.

## Security

CODA implements several security measures:

- **Sandboxed File Access**: Operations are restricted to allowed paths
- **Approval System**: Dangerous operations require explicit user consent
- **API Key Encryption**: Credentials are stored securely
- **Path Validation**: Prevents directory traversal attacks

## Development

### Prerequisites

- Go 1.21 or higher
- Make (optional, for using Makefile)

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run linter
make lint
```

### Project Structure

```
coda/
├── cmd/           # CLI commands
├── internal/      # Internal packages
│   ├── ai/       # AI client implementations
│   ├── chat/     # Chat handling logic
│   ├── config/   # Configuration management
│   ├── security/ # Security validation
│   ├── tools/    # Tool implementations
│   └── ui/       # Terminal UI (Bubbletea)
├── docs/         # Documentation
└── scripts/      # Build and utility scripts
```

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

### Development Workflow

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Troubleshooting

### Common Issues

**API Key Not Found**
```bash
# Check if API key is set
coda config get ai.api_key

# Re-set the API key
coda config set-api-key openai
```

**Permission Denied for File Operations**
- Check your `allowed_paths` configuration
- Ensure you have proper file system permissions

**Connection Timeout**
- Verify your internet connection
- Check if you're behind a proxy
- Try increasing timeout in configuration

### Debug Mode

Enable debug mode for detailed logging:

```bash
coda --debug chat
```

Check logs at `~/.coda/coda.log`

## License

MIT License

Copyright (c) 2024 Common Creation

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- UI powered by [Bubbletea](https://github.com/charmbracelet/bubbletea)
- AI integration via OpenAI and Azure OpenAI APIs

## Roadmap

- [x] Basic chat functionality
- [x] File operation tools
- [x] Multi-model support
- [x] Configuration management
- [ ] Rich terminal UI
- [ ] Plugin system
- [ ] Local model support
- [ ] Team collaboration features

---

<div align="center">
Made with ❤️ by the CODA team
</div>