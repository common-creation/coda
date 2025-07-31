# CODA Troubleshooting Guide

## Common Issues and Solutions

### Installation Issues

#### Issue: "command not found: coda"

**Cause**: CODA binary is not in your PATH.

**Solutions**:

1. **If installed via Go install**:
   ```bash
   # Check GOPATH
   echo $GOPATH
   
   # Add Go bin to PATH
   export PATH=$PATH:$(go env GOPATH)/bin
   
   # Make permanent (Linux/macOS)
   echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
   source ~/.bashrc
   ```

2. **If installed manually**:
   ```bash
   # Move to system PATH
   sudo mv coda /usr/local/bin/
   
   # Or add current directory to PATH
   export PATH=$PATH:$(pwd)
   ```

3. **Windows PowerShell**:
   ```powershell
   # Add to user PATH
   $env:PATH += ";C:\path\to\coda"
   
   # Or add via System Properties
   ```

#### Issue: Permission denied when running coda

**Solutions**:
```bash
# Make executable (Linux/macOS)
chmod +x coda

# Run with sudo if needed
sudo coda

# Change ownership
sudo chown $USER:$USER coda
```

### Configuration Issues

#### Issue: "Failed to load configuration"

**Causes and Solutions**:

1. **Invalid YAML syntax**:
   ```bash
   # Validate YAML
   coda config validate
   
   # Reset to defaults
   coda config reset
   ```

2. **Missing config file**:
   ```bash
   # Initialize default config
   coda config init
   ```

3. **Permission issues**:
   ```bash
   # Fix permissions
   chmod 600 ~/.config/coda/config.yaml
   
   # Check directory permissions
   ls -la ~/.config/coda/
   ```

#### Issue: "Invalid API key"

**Solutions**:
```bash
# Set API key correctly
coda config set openai.api_key "sk-your-actual-key"

# Verify configuration
coda config show

# Test API connection
coda config test-connection
```

### Runtime Issues

#### Issue: "Connection timeout" or "Network error"

**Diagnostic Steps**:
```bash
# Check internet connectivity
ping api.openai.com

# Test with curl
curl -H "Authorization: Bearer YOUR_KEY" https://api.openai.com/v1/models

# Check proxy settings
echo $HTTP_PROXY
echo $HTTPS_PROXY
```

**Solutions**:

1. **Proxy configuration**:
   ```bash
   # Set proxy in config
   coda config set proxy.http "http://proxy.company.com:8080"
   coda config set proxy.https "http://proxy.company.com:8080"
   
   # Or use environment variables
   export HTTP_PROXY=http://proxy.company.com:8080
   export HTTPS_PROXY=http://proxy.company.com:8080
   ```

2. **Firewall issues**:
   - Ensure ports 80 and 443 are open
   - Add CODA to firewall exceptions
   - Check corporate network restrictions

3. **DNS issues**:
   ```bash
   # Try different DNS
   echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
   ```

#### Issue: Application crashes or hangs

**Diagnostic Steps**:
```bash
# Run with debug output
coda chat --debug

# Check system resources
top
df -h

# Check for crash logs
ls ~/.config/coda/logs/
```

**Solutions**:

1. **Memory issues**:
   - Close other applications
   - Increase swap space
   - Use lighter model (gpt-3.5-turbo)

2. **Corrupted session**:
   ```bash
   # Clear sessions
   rm -rf ~/.config/coda/sessions/
   
   # Start fresh
   coda chat
   ```

### UI Issues

#### Issue: Display corruption or strange characters

**Causes and Solutions**:

1. **Terminal compatibility**:
   ```bash
   # Check terminal type
   echo $TERM
   
   # Set proper terminal
   export TERM=xterm-256color
   ```

2. **Color support**:
   ```bash
   # Test color support
   curl -s https://gist.githubusercontent.com/lifepillar/09a44b8cf0f9397465614e622979107f/raw/24-bit-color.sh | bash
   
   # Disable colors if needed
   coda config set ui.disable_colors true
   ```

3. **Font issues**:
   - Use a monospace font
   - Ensure Unicode support
   - Try different terminal emulators

#### Issue: Keyboard shortcuts not working

**Solutions**:
```bash
# Check key binding style
coda config show | grep key_bindings

# Reset to defaults
coda config set ui.key_bindings "default"

# Or try different style
coda config set ui.key_bindings "vim"
```

### Tool Execution Issues

#### Issue: "Permission denied" for file operations

**Solutions**:
```bash
# Check file permissions
ls -la target_file

# Change permissions
chmod 644 target_file

# Change ownership
sudo chown $USER:$USER target_file

# Run from correct directory
cd /path/to/project
```

#### Issue: Tools not found or not working

**Diagnostic Steps**:
```bash
# List available tools
coda tools list

# Check tool configuration
coda config show | grep tools
```

**Solutions**:
1. **Tool registration issues**:
   ```bash
   # Restart CODA to reload tools
   coda chat
   ```

2. **Path issues**:
   ```bash
   # Ensure CODA is in project directory
   pwd
   ls -la
   ```

### API Issues

#### Issue: Rate limiting errors

**Solutions**:
```bash
# Configure rate limits
coda config set openai.requests_per_minute 20
coda config set openai.retry_delay "2s"

# Use different model tier
coda config set ai.default_model "gpt-3.5-turbo"
```

#### Issue: Token limit exceeded

**Solutions**:
```bash
# Reduce max tokens
coda config set openai.max_tokens 2000

# Clear conversation history
:clear

# Use conversation summary
coda config set chat.auto_summarize true
```

## Diagnostic Commands

### System Information
```bash
# CODA version and build info
coda version

# System diagnostics
coda doctor

# Configuration validation
coda config validate

# Show current configuration
coda config show
```

### Debug Mode
```bash
# Enable debug logging
coda chat --debug

# View logs
tail -f ~/.config/coda/logs/coda.log

# Check log levels
coda config set logging.level "debug"

# Enable advanced debug mode
:debug toggle

# Set debug level (basic, detailed, verbose, trace)
:debug level verbose

# Show debug information panel
:debug panel

# Dump complete application state
:debug dump

# Enable distributed tracing
:debug trace on
```

### Network Testing
```bash
# Test API connectivity
coda config test-connection

# Check API endpoints
curl -I https://api.openai.com/v1/models
```

## Advanced Debugging and Logging

### Debug Information Panel

When debug mode is enabled, you can access detailed runtime information:

```bash
# In chat session
:debug toggle    # Enable debug mode
:debug panel     # Show debug information panel
```

The debug panel shows:
- **Uptime**: How long CODA has been running
- **Memory Usage**: Current and system memory allocation
- **Goroutines**: Number of active goroutines
- **HTTP Requests**: API call statistics
- **Custom Metrics**: Application-specific debug data

### Distributed Tracing

For performance analysis and issue investigation:

```bash
# Enable tracing
:debug trace on

# View trace visualization
:debug trace show <trace_id>

# Export traces to file
coda config set debug.trace_export_file "/tmp/coda_traces.json"
```

**Trace Features**:
- **Span Hierarchy**: Shows parent-child relationships between operations
- **Duration Tracking**: Identifies performance bottlenecks
- **Error Tracking**: Highlights failed operations
- **Context Propagation**: Tracks request flow through system

### Structured Logging Configuration

CODA uses structured logging with multiple outputs and formats:

```yaml
# ~/.config/coda/config.yaml
logging:
  level: "debug"  # debug, info, warn, error
  outputs:
    - type: "console"
      format: "text"
      options:
        colorize: true
    - type: "file"
      target: "logs/app.log"
      format: "json"
  
  # Privacy protection
  privacy:
    enabled: true
    sensitive_keys: ["api_key", "token", "password"]
    
  # Log rotation
  rotation:
    max_size: 104857600  # 100MB
    max_age: "168h"      # 7 days
    compression: true
```

### Log Sampling and Performance

For high-volume environments:

```yaml
logging:
  sampling:
    enabled: true
    rate: 0.1          # Sample 10% of debug logs
    burst_limit: 100
    burst_window: "1m"
    
  buffering:
    enabled: true
    size: 4096
    flush_level: "error"
    flush_interval: "5s"
```

### Debugging Memory Issues

```bash
# Monitor memory usage in debug panel
:debug panel

# Check for memory leaks
:debug dump | grep -i memory

# Enable GC debugging
export GODEBUG=gctrace=1
coda chat --debug
```

### Debugging Performance Issues

```bash
# Enable profiling
:debug profile start

# After operation
:debug profile stop

# View hot functions
:debug dump | grep -A 10 "hot_functions"

# Check for blocking operations
:debug dump | grep -A 10 "blocking_profile"
```

### Custom Debug Collectors

For application-specific debugging, CODA supports custom data collectors:

**Built-in collectors**:
- Session information
- Cache statistics  
- Tool execution history
- API request/response details

## Log Analysis

### Advanced Log Messages

#### Debug and Tracing Related

**"Debug mode enabled"**
- **Info**: Advanced debugging features are active
- **Impact**: Increased memory usage and log verbosity

**"Trace exported"**  
- **Info**: Distributed trace has been exported
- **File Location**: Check configured trace export location

**"Profiling started/stopped"**
- **Info**: Performance profiling is active/inactive
- **Usage**: Use :debug dump to view profile data

**"State inspection failed"**
- **Cause**: System state collector encountered error
- **Solution**: Check file permissions and system resources

#### Structured Logging Issues

**"Logger output error"**
- **Cause**: Log file write failure or disk full
- **Solution**: Check disk space and log file permissions

**"Log rotation failed"**
- **Cause**: Cannot rotate log files
- **Solution**: Ensure log directory is writable

**"Sanitizer error"**
- **Cause**: Error in privacy protection sanitization
- **Impact**: Sensitive data might be logged

### Common Log Messages

#### "context deadline exceeded"
- **Cause**: Request timeout
- **Solution**: Increase timeout values or check network

#### "invalid character 'i' looking for beginning of value"
- **Cause**: Malformed JSON response
- **Solution**: Check API endpoint configuration

#### "tool execution failed"
- **Cause**: Tool permission or path issues
- **Solution**: Check file permissions and paths

### Log Locations

**Linux/macOS**:
```
~/.config/coda/logs/
├── coda.log          # Main application log
├── api.log           # AI API interactions
├── tools.log         # Tool execution log
└── ui.log            # UI events
```

**Windows**:
```
%APPDATA%\coda\logs\
```

## Performance Issues

### Slow Responses

**Diagnostic**:
```bash
# Check network latency
ping api.openai.com

# Monitor API response times
coda chat --debug | grep "response_time"
```

**Solutions**:
1. Use faster model (gpt-3.5-turbo vs gpt-4)
2. Reduce context length
3. Enable response caching
4. Check network quality

### High Memory Usage

**Diagnostic**:
```bash
# Monitor memory usage
top -p $(pgrep coda)

# Check conversation history size
du -sh ~/.config/coda/sessions/
```

**Solutions**:
```bash
# Clear old sessions
coda config set chat.max_history 100

# Enable automatic cleanup
coda config set chat.auto_cleanup true

# Reduce concurrent operations
coda config set tools.max_concurrent 2
```

## Environment-Specific Issues

### WSL2 (Windows Subsystem for Linux)

**Common Issues**:
- Terminal color support
- File path differences
- Network connectivity

**Solutions**:
```bash
# Fix terminal colors
export TERM=xterm-256color

# Use Windows paths correctly
cd /mnt/c/Users/username/project

# Network troubleshooting
ping 8.8.8.8
```

### Docker/Containers

**Mount volumes correctly**:
```bash
docker run -v $(pwd):/workspace -w /workspace coda:latest
```

**Network access**:
```bash
# Allow network access
docker run --network host coda:latest
```

### SSH/Remote Usage

**X11 forwarding issues**:
```bash
# Enable X11 forwarding
ssh -X user@remote

# Or disable UI acceleration
coda config set ui.hardware_acceleration false
```

## Getting Help

### Built-in Help
```bash
# General help
coda help

# Command-specific help
coda chat --help

# Interactive help
coda chat
# Then press '?' for help
```

### Community Support

1. **GitHub Issues**: [github.com/common-creation/coda/issues](https://github.com/common-creation/coda/issues)
   - Search existing issues first
   - Include debug output
   - Provide system information

2. **Discussions**: [github.com/common-creation/coda/discussions](https://github.com/common-creation/coda/discussions)
   - General questions
   - Feature requests
   - Community tips

3. **Documentation**: [docs.coda.dev](https://docs.coda.dev)
   - Complete user guide
   - API documentation
   - Video tutorials

### Reporting Issues

When reporting issues, include:

1. **System Information**:
   ```bash
   coda version
   echo $TERM
   uname -a
   ```

2. **Configuration** (remove sensitive data):
   ```bash
   coda config show
   ```

3. **Debug Output**:
   ```bash
   coda chat --debug 2>&1 | tee debug.log
   ```

4. **Steps to Reproduce**:
   - Exact commands used
   - Expected vs actual behavior
   - Error messages

5. **Log Files**:
   - Relevant log entries
   - Stack traces if available