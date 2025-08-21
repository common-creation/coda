# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CODA (CODing Agent) is a CLI-based AI coding assistant written in Go. It leverages OpenAI/Azure OpenAI models and provides a rich terminal interface using Bubbletea framework.

## Essential Commands

### Development Commands

```bash
# Build the binary
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linter (golangci-lint)
make lint

# Format code
make fmt

# Build for all platforms
make build-all

# Run the application
make run

# Clean build artifacts
make clean

# Download and verify dependencies
make deps
make verify
```

### Single Test Execution

```bash
# Run a specific test
go test -v -run TestFunctionName ./path/to/package

# Run tests in a specific package
go test -v ./internal/ai/...

# Run with coverage for specific package
go test -v -coverprofile=coverage.out ./internal/chat
```

## Architecture Overview

The project follows a layered architecture with clear separation of concerns:

### Core Layers

1. **CLI Layer** (`cmd/`) - Cobra-based command handling
2. **TUI Layer** (`internal/ui/`) - Bubbletea terminal interface (in development)
3. **Application Layer** (`internal/chat/`) - Core business logic and chat processing
4. **Service Layer** (`internal/ai/`, `internal/tools/`) - AI client abstractions and tool system
5. **Infrastructure Layer** (`internal/config/`, `internal/security/`) - Configuration and security

### Key Design Patterns

- **Interface-Based Design**: All major components use interfaces for abstraction
- **Provider Pattern**: AI providers (OpenAI/Azure) implement common `Client` interface
- **Tool System**: Extensible plugin-like architecture for file operations
- **Security First**: All file operations go through validation layer

### Critical Components

1. **AI Client** (`internal/ai/`)
   - Unified interface for multiple AI providers
   - Streaming support with channel-based communication
   - Provider implementations: `openai.go`, `azure.go`

2. **Tool System** (`internal/tools/`)
   - Tool interface for extensibility
   - Built-in tools: read_file, write_file, edit_file, list_files, search_files
   - Security validation before execution

3. **Chat Handler** (`internal/chat/`)
   - Message routing and processing
   - Tool call detection in AI responses
   - Session management and persistence

4. **Configuration** (`internal/config/`)
   - YAML-based configuration with environment variable support
   - Secure credential management
   - Multi-location config loading

## Development Guidelines

### Tool Implementation Pattern

When implementing new tools:
1. Implement the `Tool` interface in `internal/tools/`
2. Add security validation in `Execute()` method
3. Register tool in the manager
4. Add tests following table-driven pattern

### AI Provider Integration

To add a new AI provider:
1. Implement `ai.Client` interface
2. Add provider configuration structure
3. Register in client factory
4. Support both streaming and non-streaming modes

### Error Handling

- Use typed errors with `CodaError` structure
- Always wrap lower-level errors with context
- Provide user-friendly error messages
- Log detailed errors for debugging

### Testing Strategy

- Unit tests for individual components with mocks
- Integration tests for component interactions
- Table-driven tests for comprehensive coverage
- Use `testify` for assertions

## Configuration

The system loads configuration from multiple sources (in order):
1. Command line flags
2. Environment variables (CODA_* prefix)
3. `$HOME/.coda/config.yaml`
4. `./config.yaml`

### Key Configuration Areas

- **AI Settings**: Provider, model, API keys
- **Tool Settings**: Enabled tools, auto-approval, allowed paths
- **Session Settings**: History management, persistence
- **Security Settings**: Path restrictions, dangerous patterns

### GPT-5 Support

CODA now supports GPT-5 models with reasoning effort configuration:

```yaml
ai:
  model: gpt-5
  # Reasoning effort for GPT-5 models (optional)
  # Valid values: "minimal", "low", "medium", "high"
  reasoning_effort: "minimal"
```

- Use `reasoning_effort: "minimal"` for fastest responses
- Higher values (`"low"`, `"medium"`, `"high"`) provide more detailed reasoning
- If `reasoning_effort` is not specified or commented out, the SDK default will be used
- This setting only applies to GPT-5 models

Note: Full GPT-5 support depends on the go-openai SDK. Currently, the reasoning effort is prepared but may not be sent to the API until SDK support is complete.

## Project-Specific Context

### System Prompts

The system uses a comprehensive prompt system defined in `internal/chat/prompts.go`. It supports:
- Default system prompt with tool calling protocol
- Workspace-specific prompts (CODA.md/CLAUDE.md in project root)
- User-specific instructions

### Tool Calling Protocol

Tools are invoked via JSON blocks in AI responses:
```json
{"tool": "tool_name", "arguments": {"param1": "value1"}}
```

### Security Considerations

- File operations restricted to allowed paths
- Dangerous patterns detection (e.g., .env, .pem files)
- User approval required for tool execution
- API keys stored securely using OS keychain

## Common Development Tasks

### Adding a New Command

1. Create new file in `cmd/` directory
2. Define command using Cobra structure
3. Register command in `root.go`
4. Add corresponding handler logic

### Modifying Tool Behavior

1. Update tool implementation in `internal/tools/`
2. Modify security rules if needed
3. Update tests
4. Document changes in tool description

### Debugging

- Enable debug mode: `coda --debug chat`
- Check logs at `~/.coda/coda.log`
- Use structured logging with appropriate levels
- Add context to errors for better tracing