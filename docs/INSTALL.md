# CODA Installation Guide

## System Requirements

- Go 1.21+ (for building from source)
- Terminal with 256 color support (recommended)
- Internet connection for AI API access

## Quick Install

### macOS

**Using Homebrew (Recommended):**
```bash
# Coming soon
brew tap common-creation/coda
brew install coda
```

**Using Go install:**
```bash
go install github.com/common-creation/coda/cmd/coda@latest
```

### Linux

**Using install script:**
```bash
# Coming soon
curl -sSL https://get.coda.dev | bash
```

**Using Go install:**
```bash
go install github.com/common-creation/coda/cmd/coda@latest
```

### Windows

**Using Scoop:**
```powershell
# Coming soon
scoop bucket add coda https://github.com/common-creation/scoop-coda
scoop install coda
```

**Using Go install:**
```powershell
go install github.com/common-creation/coda/cmd/coda@latest
```

## Manual Installation

### Download Pre-built Binaries

1. Visit the [releases page](https://github.com/common-creation/coda/releases)
2. Download the appropriate binary for your OS/architecture
3. Extract and move to your PATH:

**Linux/macOS:**
```bash
# Extract
tar -xzf coda_linux_amd64.tar.gz

# Move to PATH
sudo mv coda /usr/local/bin/
```

**Windows:**
```powershell
# Extract to desired location
# Add to PATH via System Environment Variables
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/common-creation/coda.git
cd coda

# Build
make build

# Install (optional)
make install
```

## Initial Setup

### 1. Initialize Configuration

```bash
# Create default config file
coda config init

# This creates ~/.config/coda/config.yaml
```

### 2. Configure API Keys

**OpenAI:**
```bash
coda config set openai.api_key "your-openai-key"
```

**Azure OpenAI:**
```bash
coda config set azure.api_key "your-azure-key"
coda config set azure.endpoint "https://your-resource.openai.azure.com"
```

### 3. Test Installation

```bash
# Verify installation
coda version

# Test basic functionality
coda chat -m "Hello, CODA!"
```

## Environment-Specific Notes

### WSL2 (Windows Subsystem for Linux)

CODA works well on WSL2. Ensure your terminal supports 256 colors:

```bash
# Check color support
echo $TERM

# For better experience, use Windows Terminal
```

### SSH/Remote Usage

When using CODA over SSH:

```bash
# Enable color support
export TERM=xterm-256color

# For tmux users
export TERM=screen-256color
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o coda cmd/coda/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/coda /usr/local/bin/
ENTRYPOINT ["coda"]
```

### Corporate Proxy

Configure proxy settings:

```bash
# Set proxy environment variables
export HTTP_PROXY=http://proxy.company.com:8080
export HTTPS_PROXY=http://proxy.company.com:8080

# Or configure in CODA config
coda config set proxy.http "http://proxy.company.com:8080"
coda config set proxy.https "http://proxy.company.com:8080"
```

## Troubleshooting

### Permission Errors

**Linux/macOS:**
```bash
# If installed via Go install, ensure GOBIN is in PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Or add to shell profile
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
```

**Windows:**
- Ensure Go bin directory is in system PATH
- Run PowerShell as Administrator if needed

### Missing Dependencies

**Linux:**
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install build-essential

# CentOS/RHEL
sudo yum groupinstall "Development Tools"
```

**macOS:**
```bash
# Install Xcode command line tools
xcode-select --install
```

### Network Issues

1. Check internet connectivity
2. Verify firewall settings
3. Test API endpoints:

```bash
# Test OpenAI API
curl -H "Authorization: Bearer YOUR_KEY" https://api.openai.com/v1/models

# Test network connectivity
coda config test-connection
```

### Configuration Issues

```bash
# Reset configuration
coda config reset

# Validate configuration
coda config validate

# Show current configuration
coda config show
```

## Verification

After installation, verify everything works:

```bash
# Check version
coda version

# Validate configuration
coda config validate

# Run diagnostics
coda doctor

# Test basic chat
coda chat -m "Say hello and confirm you're working properly"
```

## Uninstalling

### Homebrew
```bash
brew uninstall coda
```

### Manual/Go install
```bash
# Remove binary
rm $(which coda)

# Remove configuration (optional)
rm -rf ~/.config/coda
```

## Getting Help

- Documentation: [docs.coda.dev](https://docs.coda.dev)
- Issues: [GitHub Issues](https://github.com/common-creation/coda/issues)
- Discussions: [GitHub Discussions](https://github.com/common-creation/coda/discussions)