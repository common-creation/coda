# CODA - CODing Assistant

![](https://i.imgur.com/stKKmbT.png)

<div align="center">

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.24-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI Status](https://github.com/common-creation/coda/workflows/CI/badge.svg)](https://github.com/common-creation/coda/actions)

An intelligent command-line coding assistant that helps create, understand, and manage code through natural language dialogue.

</div>

## Features

- 🤖 **Multi-Model Support**: Compatible with OpenAI GPT and Azure OpenAI models
- 💬 **Interactive Chat**: Natural language interface for coding tasks
- 🛠️ **Tool Integration**: Built-in file operations (read, write, edit, search ...)
- 🔒 **Security First**: Approval system to avoid unintended tool calls
- 📝 **Context Aware**: Understands project structure and dependencies
- 🎨 **Rich Terminal UI**: Beautiful interface powered by Bubbletea
- 🔧 **Configurable**: Extensive configuration options via YAML/environment variables

## Quick Start

### Installation

#### From Releases

https://github.com/common-creation/coda/releases/latest

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

or

```bash
coda
# Configuration file will be created on first launch or if API key is not set
```

2. Set API keys:
```bash
# For OpenAI
coda config set-api-key openai [key]

# For Azure OpenAI
coda config set-api-key azure [key]
```

You can also edit the configuration file directly.

3. (Optional) Customize configuration:
```bash
coda config set ai.model o4-mini # Default is o3
```

### Basic Usage

#### Interactive Chat Mode

```bash
# Start interactive chat
coda

# Use specific model
coda --model o4-mini
```

## Commands

### `coda` or `coda chat`
Start an interactive chat session with the AI assistant.

**Options:**
- `--model`: Specify the AI model to use

### `coda config`
Manage CODA configuration.

**Subcommands:**
- `show`: Display current configuration
- `set KEY VALUE`: Set a configuration value
- `get KEY`: Get a specific value
- `init`: Initialize configuration file
- `validate`: Check configuration validity
- `set-api-key PROVIDER`: Set API key without leaving it in history

### `coda version`
Display version information.

**Options:**
- `--verbose, -v`: Show detailed version information
- `--json`: Output in JSON format

## Configuration

CODA loads configuration from the following locations (in order):
1. Command-line flags: `--config`
2. Environment variables: `CODA_CONFIG`
3. `$HOME/.coda/config.yaml`
4. `./config.yaml`

### Example Configuration File

```yaml
ai:
  provider: openai  # or "azure"
  model: o3
  temperature: 1
  max_tokens: 0 # 0 means no limit

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
export CODA_AI_MODEL=o4-mini
export CODA_AI_API_KEY=sk-...
```

## Workspace Configuration

CODA supports project-specific configuration through workspace files:

### `.coda/CODA.md` or `.claude/CLAUDE.md` (experimental)

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

CODA includes several built-in tools for file operations.
Here are some representative examples:

- **read_file**: Read file contents
- **write_file**: Create or overwrite files
- **edit_file**: Modify specific parts of files
- **list_files**: List directory contents
- **search_files**: Search files by content or name

For security, all tool operations require user approval by default.

## Security

CODA implements multiple security measures:

- **Restricted File Access**: Operations are limited to allowed paths
- **Approval System**: Tool calls require explicit user consent
- **Path Validation**: Prevents directory traversal attacks

## Development

### Prerequisites

- Go 1.24 or higher

### Project Structure

```
coda/
├── cmd/           # CLI commands
├── internal/      # Internal packages
│   ├── ai/       # AI client implementations
│   ├── chat/     # Chat processing logic
│   ├── config/   # Configuration management
│   ├── security/ # Security validation
│   ├── tools/    # Tool implementations
│   └── ui/       # Terminal UI (Bubbletea)
├── docs/         # Documentation
├── scripts/      # Build and utility scripts
└── tests/        # Tests
```

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Troubleshooting

### Common Issues

**API Key Not Found**
```bash
# Check if API key is set
coda config get ai.api_key

# Reset API key
coda config set-api-key openai
```

**File Operation Permission Denied**
- Check `allowed_paths` configuration
- Ensure you have proper file system permissions

**Connection Timeout**
- Check your internet connection
- Verify if you're behind a proxy

### Debug Mode

Enable debug mode for detailed logging:

```bash
coda --debug chat
```

Logs can be found at `~/.coda/coda.log`

## License

MIT License

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- UI powered by [Bubbletea](https://github.com/charmbracelet/bubbletea)
- AI integration via OpenAI and Azure OpenAI APIs

## Roadmap

- [x] Basic chat functionality
- [x] File operation tools
- [x] Multi-model support
- [x] Configuration management
- [x] Rich terminal UI
- [ ] Additional tools
- [ ] Plugin system
- [ ] Local model support

---

<div align="center">
Made with ❤️ by the CODA team
</div>