package security

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// SecurityPatterns holds dangerous patterns for security validation
type SecurityPatterns struct {
	DangerousPaths      []string
	DangerousExtensions []string
	DangerousContent    []*regexp.Regexp
	SystemPaths         []string
	SensitiveFiles      []string
}

// GetDefaultPatterns returns the default security patterns
func GetDefaultPatterns() *SecurityPatterns {
	patterns := &SecurityPatterns{
		DangerousPaths: []string{
			// Unix/Linux system paths
			"/etc",
			"/sys",
			"/proc",
			"/dev",
			"/boot",
			"/root",
			"/usr/bin",
			"/usr/sbin",
			"/bin",
			"/sbin",
			"/var/log/secure",
			"/var/log/auth.log",

			// User home sensitive directories
			"~/.ssh",
			"~/.gnupg",
			"~/.aws",
			"~/.kube",
			"~/.docker",
			"~/.config/gcloud",
			"~/.azure",

			// Git related
			".git/config",
			".git/credentials",

			// Environment files
			".env",
			".envrc",
			".env.local",
			".env.production",
			".env.development",
		},

		DangerousExtensions: []string{
			// Executable files
			".exe", ".dll", ".so", ".dylib", ".app",
			".msi", ".deb", ".rpm", ".dmg", ".pkg",

			// Scripts that could be executed
			".sh", ".bash", ".zsh", ".fish",
			".bat", ".cmd", ".ps1", ".psm1",
			".vbs", ".js", ".jar", ".war",

			// System and config files
			".sys", ".drv", ".vxd", ".386",

			// Certificate and key files
			".pem", ".key", ".pfx", ".p12",
			".cer", ".crt", ".der",

			// Database files
			".db", ".sqlite", ".sqlite3",
			".mdb", ".accdb",
		},

		SensitiveFiles: []string{
			// SSH keys
			"id_rsa", "id_dsa", "id_ecdsa", "id_ed25519",
			"id_rsa.pub", "id_dsa.pub", "id_ecdsa.pub", "id_ed25519.pub",
			"authorized_keys", "known_hosts",

			// AWS credentials
			"credentials", "config",

			// Kubernetes
			"kubeconfig", "admin.conf",

			// Docker
			"docker/config.json",

			// Various tokens and secrets
			".npmrc", ".pypirc", ".netrc",
			".git-credentials", ".gitconfig",

			// History files
			".bash_history", ".zsh_history",
			".mysql_history", ".psql_history",
			".python_history", ".node_repl_history",

			// Password files
			"passwd", "shadow", "master.passwd",

			// macOS specific
			".DS_Store", "Keychain",

			// Windows specific
			"NTUSER.DAT", "SAM", "SYSTEM", "SECURITY",
		},
	}

	// Platform-specific paths
	switch runtime.GOOS {
	case "windows":
		patterns.SystemPaths = []string{
			"C:\\Windows",
			"C:\\Windows\\System32",
			"C:\\Windows\\SysWOW64",
			"C:\\Program Files",
			"C:\\Program Files (x86)",
			"C:\\ProgramData",
			"C:\\Users\\Administrator",
			"C:\\Users\\Default",
		}
	case "darwin": // macOS
		patterns.SystemPaths = []string{
			"/System",
			"/Library",
			"/Applications",
			"/usr/local/bin",
			"/opt/homebrew",
			"~/Library/Keychains",
			"~/Library/Application Support",
		}
	default: // Linux and others
		patterns.SystemPaths = []string{
			"/lib",
			"/lib64",
			"/usr/lib",
			"/usr/lib64",
			"/opt",
			"/usr/local/bin",
			"/usr/local/sbin",
		}
	}

	// Compile dangerous content patterns
	patterns.DangerousContent = compileDangerousContentPatterns()

	return patterns
}

// compileDangerousContentPatterns compiles regex patterns for dangerous content
func compileDangerousContentPatterns() []*regexp.Regexp {
	patternStrings := []string{
		// Shell command injection patterns
		`\$\([^)]+\)`,       // Command substitution $(...)
		"`[^`]+`",           // Command substitution `...`
		`\beval\s*\(`,       // eval function
		`\bexec\s*\(`,       // exec function
		`\bsystem\s*\(`,     // system function
		`\bpassthru\s*\(`,   // passthru function (PHP)
		`\bshell_exec\s*\(`, // shell_exec function (PHP)

		// SQL injection patterns
		`\bUNION\s+SELECT\b`,     // UNION SELECT
		`\bDROP\s+TABLE\b`,       // DROP TABLE
		`\bDELETE\s+FROM\b`,      // DELETE FROM
		`\bINSERT\s+INTO\b`,      // INSERT INTO
		`\bUPDATE\s+\w+\s+SET\b`, // UPDATE ... SET

		// Common credential patterns
		`[Pp]assword\s*[:=]\s*["'][^"']{8,}["']`,    // Password = "..."
		`[Aa]pi[_-]?[Kk]ey\s*[:=]\s*["'][^"']+["']`, // API key
		`[Ss]ecret\s*[:=]\s*["'][^"']+["']`,         // Secret = "..."
		`[Tt]oken\s*[:=]\s*["'][^"']+["']`,          // Token = "..."
		`[Aa]uth\s*[:=]\s*["'][^"']+["']`,           // Auth = "..."

		// Base64 encoded executables (simplified pattern)
		`data:application/x-executable;base64,`, // Base64 executable
		`data:application/x-msdownload;base64,`, // Base64 Windows executable

		// Script injection
		`<script[^>]*>`, // Script tags
		`javascript:`,   // JavaScript protocol
		`vbscript:`,     // VBScript protocol
		`onload\s*=`,    // Event handlers
		`onerror\s*=`,   // Event handlers
		`onclick\s*=`,   // Event handlers

		// Path traversal
		`\.\.[\\/]`,     // ../ or ..\
		`\.\.\.\.[\\/]`, // Multiple traversals
	}

	patterns := make([]*regexp.Regexp, 0, len(patternStrings))
	for _, pattern := range patternStrings {
		// Compile with case-insensitive flag where appropriate
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			// Skip invalid patterns
			continue
		}
		patterns = append(patterns, re)
	}

	return patterns
}

// IsSystemPath checks if a path is a system path
func (p *SecurityPatterns) IsSystemPath(path string) bool {
	path = strings.ToLower(path)

	for _, sysPath := range p.SystemPaths {
		if strings.HasPrefix(path, strings.ToLower(sysPath)) {
			return true
		}
	}

	return false
}

// IsDangerousPath checks if a path matches dangerous patterns
func (p *SecurityPatterns) IsDangerousPath(path string) bool {
	path = strings.ToLower(path)

	for _, dangerous := range p.DangerousPaths {
		dangerous = strings.ToLower(dangerous)
		if strings.Contains(path, dangerous) {
			return true
		}
	}

	// Check if it's a sensitive file
	baseName := strings.ToLower(filepath.Base(path))
	for _, sensitive := range p.SensitiveFiles {
		if baseName == strings.ToLower(sensitive) {
			return true
		}
	}

	return false
}

// IsDangerousExtension checks if a file has a dangerous extension
func (p *SecurityPatterns) IsDangerousExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))

	for _, dangerous := range p.DangerousExtensions {
		if ext == strings.ToLower(dangerous) {
			return true
		}
	}

	return false
}

// HasDangerousContent checks if content matches dangerous patterns
func (p *SecurityPatterns) HasDangerousContent(content string) (bool, string) {
	for _, pattern := range p.DangerousContent {
		if pattern.MatchString(content) {
			return true, pattern.String()
		}
	}

	return false, ""
}

// ValidateFilename checks if a filename is safe
func (p *SecurityPatterns) ValidateFilename(filename string) error {
	// Check for null bytes
	if strings.Contains(filename, "\x00") {
		return fmt.Errorf("filename contains null bytes")
	}

	// Check for dangerous characters
	dangerousChars := []string{"|", "&", ";", "$", "`", "\\n", "\\r"}
	for _, char := range dangerousChars {
		if strings.Contains(filename, char) {
			return fmt.Errorf("filename contains dangerous character: %s", char)
		}
	}

	// Check if it's a sensitive file
	baseName := filepath.Base(filename)
	for _, sensitive := range p.SensitiveFiles {
		if strings.EqualFold(baseName, sensitive) {
			return fmt.Errorf("access to sensitive file '%s' is restricted", baseName)
		}
	}

	// Check dangerous extensions
	if p.IsDangerousExtension(filename) {
		return fmt.Errorf("file extension '%s' is not allowed", filepath.Ext(filename))
	}

	return nil
}
