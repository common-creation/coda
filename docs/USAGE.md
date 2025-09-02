# CODA Usage Guide

## Getting Started

### Your First Chat Session

Start CODA in any directory:

```bash
cd your-project
coda chat
```

The TUI (Terminal User Interface) will launch with:
- **Chat area**: Displays conversation history
- **Input area**: Where you type messages
- **Status bar**: Shows current mode and shortcuts
- **Help**: Press `?` for keyboard shortcuts

### Understanding the UI

CODA uses a Vim-inspired modal interface:

- **Normal Mode**: Navigate and execute commands (`ESC`)
- **Insert Mode**: Type messages (`i`)
- **Command Mode**: Execute special commands (`:`)
- **Search Mode**: Search chat history (`/`)

### Basic Commands

```bash
# Start interactive chat
coda chat

# Send a single message
coda chat -m "Explain this code" file.go

# Use specific model
coda chat --model gpt-4

# Debug mode for troubleshooting
coda chat --debug

# Non-interactive mode
coda chat --non-interactive -m "Hello"
```

## Working with Files

### Reading Files

CODA can read and analyze files in your project:

```
# In chat session
Please read and explain main.go

# CODA will use the read_file tool automatically
```

**Supported file types:**
- Source code (Go, Python, JavaScript, etc.)
- Configuration files (YAML, JSON, TOML)
- Documentation (Markdown, text)
- Data files (CSV, JSON)

### Editing Files

Request file modifications:

```
# Example requests
Can you add error handling to the login function in auth.go?
Please create a unit test for the UserService
Add documentation comments to all public functions
```

CODA will:
1. Read the relevant files
2. Analyze the code
3. Ask for confirmation before making changes
4. Show you the diff before applying

### Creating New Files

```
# Example requests
Create a new REST API endpoint for user management
Generate a Docker configuration for this Go project
Create a comprehensive README for this project
```

## Advanced Features

### Tool Execution

CODA has built-in tools for development tasks:

- **read_file**: Read file contents
- **write_file**: Create new files
- **edit_file**: Modify existing files
- **list_files**: Show directory structure
- **search_files**: Find files by content

**Tool Approval System:**
- CODA asks permission before executing potentially dangerous operations
- You can approve/deny each tool execution
- Configure auto-approval for trusted operations

### Session Management

```bash
# Save current session
:save session-name

# Load previous session
:load session-name

# List saved sessions
:sessions

# Clear current chat
:clear
```

### Workspace Configuration

Create a `CODA.md` file in your project root for custom instructions:

```markdown
# Project: My Web Application

## Context
This is a Go web application using Gin framework and PostgreSQL.

## Guidelines
- Always include error handling
- Follow Go naming conventions
- Add tests for new features
- Update documentation for API changes

## File Structure
- `/cmd` - Application entry points
- `/internal` - Private application code  
- `/pkg` - Public library code
- `/api` - API definitions
```

## Keyboard Shortcuts

### Global Shortcuts
- `Ctrl+C` / `Ctrl+D`: Quit application
- `?` / `F1`: Show/hide help
- `Ctrl+L`: Clear screen
- `F5` / `Ctrl+R`: Refresh view

### Normal Mode
- `i`: Enter insert mode
- `:`: Enter command mode
- `/`: Search forward
- `?`: Search backward
- `k`/`↑`: Scroll up
- `j`/`↓`: Scroll down
- `Enter`: Send message (if input not empty)

### Insert Mode
- `ESC`: Exit to normal mode
- `Enter`: Send message
- `Ctrl+S`: Save and exit
- `Ctrl+C`: Force exit

### Command Mode
- `ESC`: Exit to normal mode
- `Enter`: Execute command
- `Tab`: Command completion

### Search Mode
- `ESC`: Exit search
- `Enter`: Execute search
- `n`: Next match
- `N`: Previous match

## Productivity Tips

### Effective Prompts

**Good prompts:**
- "Review the authentication logic in auth.go and suggest improvements"
- "Add comprehensive error handling to the database connection code"
- "Create a unit test for the calculateTotal function with edge cases"

**Less effective:**
- "Fix this" (too vague)
- "Make it better" (unclear requirements)
- "Debug" (no specific context)

### Command Aliases

Common command shortcuts:
- `:q` - Quit
- `:h` - Help
- `:clear` - Clear chat history
- `:new` - Start new chat session

### Workflow Examples

#### Code Review Workflow

1. Start CODA in project directory:
   ```bash
   cd my-project
   coda chat
   ```

2. Request comprehensive review:
   ```
   Please review the code in src/handlers/ directory. Focus on:
   - Error handling
   - Input validation  
   - Security concerns
   - Code organization
   ```

3. Address feedback iteratively:
   ```
   Can you show me how to improve the validation in user_handler.go?
   ```

#### Bug Fixing Workflow

1. Describe the issue:
   ```
   I'm getting a panic in my HTTP handler when processing POST requests.
   The error occurs in handlers/api.go around line 45.
   ```

2. Let CODA analyze:
   ```
   Please read handlers/api.go and identify potential causes for the panic
   ```

3. Apply fixes:
   ```
   Can you add proper nil checking and error handling to prevent this panic?
   ```

#### Feature Development

1. Describe requirements:
   ```
   I need to add user authentication to my web app. It should support:
   - JWT tokens
   - Password hashing
   - Login/logout endpoints
   - Middleware for protected routes
   ```

2. Let CODA plan the implementation:
   ```
   Please outline the files and functions needed for this authentication system
   ```

3. Implement step by step:
   ```
   Let's start with the JWT token generation function
   ```

### Performance Optimization

- Use specific file paths when possible
- Break large requests into smaller, focused ones
- Save and load sessions for complex projects
- Use search to quickly find previous discussions

## Configuration Tips

### Model Selection

```yaml
# ~/.config/coda/config.yaml
ai:
  default_model: "o3"
  fallback_model: "gpt-3.5-turbo"
  
models:
  gpt-4:
    max_tokens: 4000
    temperature: 0.1
  gpt-3.5-turbo:
    max_tokens: 2000
    temperature: 0.2
```

### UI Customization

```yaml
ui:
  theme: "dark"  # light, dark, auto
  key_bindings: "vim"  # vim, emacs, default
  enable_mouse: true
  show_line_numbers: true
```

### Tool Configuration

```yaml
tools:
  auto_approve:
    - "read_file"
    - "list_files"
  require_approval:
    - "write_file"
    - "edit_file"
  security:
    max_file_size: "10MB"
    allowed_extensions: [".go", ".js", ".py", ".md"]
```

## Best Practices

### Security
- Review all file modifications before approval
- Be cautious with auto-approval settings
- Keep API keys secure
- Don't commit sensitive information

### Organization
- Use descriptive session names
- Organize projects with CODA.md files
- Keep conversations focused
- Archive old sessions periodically

### Collaboration
- Share CODA.md configurations with team
- Document project-specific prompts
- Use consistent coding standards
- Review AI-generated code thoroughly

## Troubleshooting

### Common Issues

**CODA not responding:**
- Check internet connection
- Verify API key configuration
- Try with `--debug` flag

**File operations failing:**
- Check file permissions
- Verify file paths
- Ensure files aren't locked by other processes

**UI display issues:**
- Ensure terminal supports 256 colors
- Try resizing terminal window
- Check TERM environment variable

### Getting Help

```bash
# Built-in help
coda help
coda chat --help

# Diagnostics
coda doctor

# Configuration check
coda config validate

# Version information
coda version
```

For more help:
- Documentation: [docs.coda.dev](https://docs.coda.dev)
- GitHub Issues: [github.com/common-creation/coda/issues](https://github.com/common-creation/coda/issues)
- Community: [github.com/common-creation/coda/discussions](https://github.com/common-creation/coda/discussions)