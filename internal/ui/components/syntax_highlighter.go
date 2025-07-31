package components

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/common-creation/coda/internal/styles"
)

// TokenType represents different types of code tokens
type TokenType int

const (
	TokenText TokenType = iota
	TokenKeyword
	TokenString
	TokenComment
	TokenFunction
	TokenNumber
	TokenOperator
	TokenType_
	TokenVariable
	TokenBracket
	TokenDelimiter
)

// Token represents a syntax token
type Token struct {
	Type    TokenType
	Content string
	Start   int
	End     int
}

// HighlightedLine represents a line of highlighted code
type HighlightedLine struct {
	Tokens     []Token
	LineNumber int
	Content    string
}

// HighlightedCode represents highlighted source code
type HighlightedCode struct {
	Language string
	Lines    []HighlightedLine
	Theme    HighlightTheme
	Raw      string
}

// HighlightTheme contains styling for different token types
type HighlightTheme struct {
	Keyword    lipgloss.Style
	String     lipgloss.Style
	Comment    lipgloss.Style
	Function   lipgloss.Style
	Number     lipgloss.Style
	Operator   lipgloss.Style
	Type       lipgloss.Style
	Variable   lipgloss.Style
	Bracket    lipgloss.Style
	Delimiter  lipgloss.Style
	Background lipgloss.Style
}

// Language contains language-specific syntax rules
type Language struct {
	Name            string
	Keywords        []string
	Operators       []string
	Types           []string
	StringDelims    []string
	CommentSingle   string
	CommentMulti    [2]string
	FunctionPattern *regexp.Regexp
	NumberPattern   *regexp.Regexp
	VariablePattern *regexp.Regexp
}

// SyntaxHighlighter provides syntax highlighting functionality
type SyntaxHighlighter struct {
	theme     HighlightTheme
	languages map[string]Language
	cache     map[string]HighlightedCode
	mutex     sync.RWMutex
}

// NewSyntaxHighlighter creates a new syntax highlighter
func NewSyntaxHighlighter(styles styles.Styles) *SyntaxHighlighter {
	sh := &SyntaxHighlighter{
		theme:     createHighlightTheme(styles),
		languages: make(map[string]Language),
		cache:     make(map[string]HighlightedCode),
	}

	sh.initializeLanguages()
	return sh
}

// createHighlightTheme creates a syntax highlight theme from UI styles
func createHighlightTheme(styles styles.Styles) HighlightTheme {
	return HighlightTheme{
		Keyword:    styles.Bold.Foreground(lipgloss.Color("#569CD6")),                      // Blue
		String:     lipgloss.NewStyle().Foreground(lipgloss.Color("#CE9178")),              // Orange
		Comment:    lipgloss.NewStyle().Foreground(lipgloss.Color("#608B4E")).Italic(true), // Green
		Function:   lipgloss.NewStyle().Foreground(lipgloss.Color("#DCDCAA")),              // Yellow
		Number:     lipgloss.NewStyle().Foreground(lipgloss.Color("#B5CEA8")),              // Light Green
		Operator:   lipgloss.NewStyle().Foreground(lipgloss.Color("#D4D4D4")),              // Light Gray
		Type:       lipgloss.NewStyle().Foreground(lipgloss.Color("#4EC9B0")),              // Cyan
		Variable:   lipgloss.NewStyle().Foreground(lipgloss.Color("#9CDCFE")),              // Light Blue
		Bracket:    lipgloss.NewStyle().Foreground(lipgloss.Color("#DA70D6")),              // Magenta
		Delimiter:  lipgloss.NewStyle().Foreground(lipgloss.Color("#D4D4D4")),              // Light Gray
		Background: lipgloss.NewStyle().Background(lipgloss.Color("#1E1E1E")),              // Dark Background
	}
}

// initializeLanguages sets up language definitions
func (sh *SyntaxHighlighter) initializeLanguages() {
	// Go language
	sh.languages["go"] = Language{
		Name: "Go",
		Keywords: []string{
			"break", "case", "chan", "const", "continue", "default", "defer",
			"else", "fallthrough", "for", "func", "go", "goto", "if", "import",
			"interface", "map", "package", "range", "return", "select", "struct",
			"switch", "type", "var",
		},
		Types: []string{
			"bool", "byte", "complex64", "complex128", "error", "float32", "float64",
			"int", "int8", "int16", "int32", "int64", "rune", "string",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		},
		Operators:       []string{"+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>", "&^", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", "&^=", "&&", "||", "<-", "++", "--", "==", "<", ">", "=", "!", "!=", "<=", ">=", ":=", "...", "(", ")", "[", "]", "{", "}", ",", ";"},
		StringDelims:    []string{`"`, "`"},
		CommentSingle:   "//",
		CommentMulti:    [2]string{"/*", "*/"},
		FunctionPattern: regexp.MustCompile(`\b(\w+)\s*\(`),
		NumberPattern:   regexp.MustCompile(`\b\d+(\.\d+)?\b`),
		VariablePattern: regexp.MustCompile(`\b[a-zA-Z_]\w*\b`),
	}

	// Python language
	sh.languages["python"] = Language{
		Name: "Python",
		Keywords: []string{
			"and", "as", "assert", "break", "class", "continue", "def", "del",
			"elif", "else", "except", "finally", "for", "from", "global", "if",
			"import", "in", "is", "lambda", "nonlocal", "not", "or", "pass",
			"raise", "return", "try", "while", "with", "yield", "async", "await",
		},
		Types: []string{
			"bool", "int", "float", "complex", "str", "bytes", "bytearray",
			"list", "tuple", "range", "dict", "set", "frozenset",
		},
		Operators:       []string{"+", "-", "*", "/", "//", "%", "**", "&", "|", "^", "~", "<<", ">>", "<", ">", "<=", ">=", "==", "!=", "=", "+=", "-=", "*=", "/=", "//=", "%=", "**=", "&=", "|=", "^=", "<<=", ">>="},
		StringDelims:    []string{`"`, `'`, `"""`, `'''`},
		CommentSingle:   "#",
		FunctionPattern: regexp.MustCompile(`\bdef\s+(\w+)\s*\(`),
		NumberPattern:   regexp.MustCompile(`\b\d+(\.\d+)?\b`),
		VariablePattern: regexp.MustCompile(`\b[a-zA-Z_]\w*\b`),
	}

	// JavaScript language
	sh.languages["javascript"] = Language{
		Name: "JavaScript",
		Keywords: []string{
			"async", "await", "break", "case", "catch", "class", "const", "continue",
			"debugger", "default", "delete", "do", "else", "export", "extends",
			"finally", "for", "function", "if", "import", "in", "instanceof",
			"let", "new", "return", "super", "switch", "this", "throw", "try",
			"typeof", "var", "void", "while", "with", "yield",
		},
		Types: []string{
			"boolean", "number", "string", "object", "undefined", "null", "symbol", "bigint",
		},
		Operators:       []string{"+", "-", "*", "/", "%", "**", "&", "|", "^", "~", "<<", ">>", ">>>", "<", ">", "<=", ">=", "==", "===", "!=", "!==", "=", "+=", "-=", "*=", "/=", "%=", "**=", "&=", "|=", "^=", "<<=", ">>=", ">>>=", "&&", "||", "!", "?", ":"},
		StringDelims:    []string{`"`, `'`, "`"},
		CommentSingle:   "//",
		CommentMulti:    [2]string{"/*", "*/"},
		FunctionPattern: regexp.MustCompile(`\b(\w+)\s*\(`),
		NumberPattern:   regexp.MustCompile(`\b\d+(\.\d+)?\b`),
		VariablePattern: regexp.MustCompile(`\b[a-zA-Z_$]\w*\b`),
	}

	// Add aliases
	sh.languages["js"] = sh.languages["javascript"]
	sh.languages["typescript"] = sh.languages["javascript"]
	sh.languages["ts"] = sh.languages["javascript"]

	// Rust language
	sh.languages["rust"] = Language{
		Name: "Rust",
		Keywords: []string{
			"as", "break", "const", "continue", "crate", "else", "enum", "extern",
			"false", "fn", "for", "if", "impl", "in", "let", "loop", "match",
			"mod", "move", "mut", "pub", "ref", "return", "self", "Self", "static",
			"struct", "super", "trait", "true", "type", "unsafe", "use", "where", "while",
		},
		Types: []string{
			"bool", "char", "i8", "i16", "i32", "i64", "i128", "isize",
			"u8", "u16", "u32", "u64", "u128", "usize", "f32", "f64", "str", "String",
		},
		Operators:       []string{"+", "-", "*", "/", "%", "&", "|", "^", "!", "<<", ">>", "&&", "||", "<", ">", "<=", ">=", "==", "!=", "=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>="},
		StringDelims:    []string{`"`, `'`},
		CommentSingle:   "//",
		CommentMulti:    [2]string{"/*", "*/"},
		FunctionPattern: regexp.MustCompile(`\bfn\s+(\w+)\s*\(`),
		NumberPattern:   regexp.MustCompile(`\b\d+(\.\d+)?\b`),
		VariablePattern: regexp.MustCompile(`\b[a-zA-Z_]\w*\b`),
	}

	// JSON (simplified)
	sh.languages["json"] = Language{
		Name:          "JSON",
		Keywords:      []string{"true", "false", "null"},
		StringDelims:  []string{`"`},
		NumberPattern: regexp.MustCompile(`\b-?\d+(\.\d+)?([eE][+-]?\d+)?\b`),
	}

	// YAML (simplified)
	sh.languages["yaml"] = Language{
		Name:          "YAML",
		Keywords:      []string{"true", "false", "null", "yes", "no"},
		StringDelims:  []string{`"`, `'`},
		CommentSingle: "#",
		NumberPattern: regexp.MustCompile(`\b-?\d+(\.\d+)?\b`),
	}

	// Shell/Bash
	sh.languages["bash"] = Language{
		Name: "Bash",
		Keywords: []string{
			"if", "then", "else", "elif", "fi", "case", "esac", "for", "while",
			"until", "do", "done", "function", "return", "local", "export",
			"unset", "readonly", "declare", "typeset", "let", "eval", "exec",
		},
		StringDelims:  []string{`"`, `'`},
		CommentSingle: "#",
		NumberPattern: regexp.MustCompile(`\b\d+\b`),
	}

	sh.languages["shell"] = sh.languages["bash"]
	sh.languages["sh"] = sh.languages["bash"]
}

// Highlight highlights code and returns highlighted representation
func (sh *SyntaxHighlighter) Highlight(code, language string) HighlightedCode {
	// Check cache first
	cacheKey := language + ":" + code
	sh.mutex.RLock()
	if cached, exists := sh.cache[cacheKey]; exists {
		sh.mutex.RUnlock()
		return cached
	}
	sh.mutex.RUnlock()

	// Perform highlighting
	result := sh.highlightCode(code, language)

	// Cache the result
	sh.mutex.Lock()
	sh.cache[cacheKey] = result
	// Limit cache size
	if len(sh.cache) > 1000 {
		// Clear half the cache
		count := 0
		for k := range sh.cache {
			delete(sh.cache, k)
			count++
			if count >= 500 {
				break
			}
		}
	}
	sh.mutex.Unlock()

	return result
}

// highlightCode performs the actual syntax highlighting
func (sh *SyntaxHighlighter) highlightCode(code, language string) HighlightedCode {
	lang, exists := sh.languages[strings.ToLower(language)]
	if !exists {
		// Return unhighlighted code
		return sh.createPlainHighlight(code, language)
	}

	lines := strings.Split(code, "\n")
	highlightedLines := make([]HighlightedLine, len(lines))

	for i, line := range lines {
		tokens := sh.tokenizeLine(line, lang)
		highlightedLines[i] = HighlightedLine{
			Tokens:     tokens,
			LineNumber: i + 1,
			Content:    line,
		}
	}

	return HighlightedCode{
		Language: language,
		Lines:    highlightedLines,
		Theme:    sh.theme,
		Raw:      code,
	}
}

// tokenizeLine tokenizes a single line of code
func (sh *SyntaxHighlighter) tokenizeLine(line string, lang Language) []Token {
	if line == "" {
		return []Token{}
	}

	var tokens []Token
	i := 0

	for i < len(line) {
		// Skip whitespace
		if line[i] == ' ' || line[i] == '\t' {
			start := i
			for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
				i++
			}
			tokens = append(tokens, Token{
				Type:    TokenText,
				Content: line[start:i],
				Start:   start,
				End:     i,
			})
			continue
		}

		// Check for comments
		if lang.CommentSingle != "" && strings.HasPrefix(line[i:], lang.CommentSingle) {
			// Single line comment - consume rest of line
			tokens = append(tokens, Token{
				Type:    TokenComment,
				Content: line[i:],
				Start:   i,
				End:     len(line),
			})
			break
		}

		// Check for multi-line comments
		if lang.CommentMulti[0] != "" && strings.HasPrefix(line[i:], lang.CommentMulti[0]) {
			start := i
			i += len(lang.CommentMulti[0])

			// Find end of comment (simplified - doesn't handle multi-line)
			end := strings.Index(line[i:], lang.CommentMulti[1])
			if end != -1 {
				i += end + len(lang.CommentMulti[1])
			} else {
				i = len(line)
			}

			tokens = append(tokens, Token{
				Type:    TokenComment,
				Content: line[start:i],
				Start:   start,
				End:     i,
			})
			continue
		}

		// Check for strings
		stringFound := false
		for _, delim := range lang.StringDelims {
			if strings.HasPrefix(line[i:], delim) {
				start := i
				i += len(delim)

				// Find closing delimiter
				for i < len(line) {
					if strings.HasPrefix(line[i:], delim) {
						i += len(delim)
						break
					}
					if line[i] == '\\' && i+1 < len(line) {
						i += 2 // Skip escaped character
					} else {
						i++
					}
				}

				tokens = append(tokens, Token{
					Type:    TokenString,
					Content: line[start:i],
					Start:   start,
					End:     i,
				})
				stringFound = true
				break
			}
		}
		if stringFound {
			continue
		}

		// Check for numbers
		if lang.NumberPattern != nil {
			if match := lang.NumberPattern.FindStringIndex(line[i:]); match != nil && match[0] == 0 {
				end := i + match[1]
				tokens = append(tokens, Token{
					Type:    TokenNumber,
					Content: line[i:end],
					Start:   i,
					End:     end,
				})
				i = end
				continue
			}
		}

		// Check for operators
		operatorFound := false
		for _, op := range lang.Operators {
			if strings.HasPrefix(line[i:], op) {
				tokenType := TokenOperator
				if op == "(" || op == ")" || op == "[" || op == "]" || op == "{" || op == "}" {
					tokenType = TokenBracket
				} else if op == "," || op == ";" || op == ":" {
					tokenType = TokenDelimiter
				}

				tokens = append(tokens, Token{
					Type:    tokenType,
					Content: op,
					Start:   i,
					End:     i + len(op),
				})
				i += len(op)
				operatorFound = true
				break
			}
		}
		if operatorFound {
			continue
		}

		// Check for identifiers (keywords, types, functions, variables)
		if lang.VariablePattern != nil {
			if match := lang.VariablePattern.FindStringIndex(line[i:]); match != nil && match[0] == 0 {
				end := i + match[1]
				word := line[i:end]

				tokenType := TokenVariable

				// Check if it's a keyword
				for _, keyword := range lang.Keywords {
					if word == keyword {
						tokenType = TokenKeyword
						break
					}
				}

				// Check if it's a type
				if tokenType == TokenVariable {
					for _, typeName := range lang.Types {
						if word == typeName {
							tokenType = TokenType_
							break
						}
					}
				}

				// Check if it's a function (simple check)
				if tokenType == TokenVariable && end < len(line) && line[end] == '(' {
					tokenType = TokenFunction
				}

				tokens = append(tokens, Token{
					Type:    tokenType,
					Content: word,
					Start:   i,
					End:     end,
				})
				i = end
				continue
			}
		}

		// Default: single character as text
		tokens = append(tokens, Token{
			Type:    TokenText,
			Content: string(line[i]),
			Start:   i,
			End:     i + 1,
		})
		i++
	}

	return tokens
}

// createPlainHighlight creates unhighlighted code representation
func (sh *SyntaxHighlighter) createPlainHighlight(code, language string) HighlightedCode {
	lines := strings.Split(code, "\n")
	highlightedLines := make([]HighlightedLine, len(lines))

	for i, line := range lines {
		tokens := []Token{{
			Type:    TokenText,
			Content: line,
			Start:   0,
			End:     len(line),
		}}

		highlightedLines[i] = HighlightedLine{
			Tokens:     tokens,
			LineNumber: i + 1,
			Content:    line,
		}
	}

	return HighlightedCode{
		Language: language,
		Lines:    highlightedLines,
		Theme:    sh.theme,
		Raw:      code,
	}
}

// Render renders highlighted code to a string
func (sh *SyntaxHighlighter) Render(highlighted HighlightedCode, showLineNumbers bool) string {
	var result strings.Builder

	for _, line := range highlighted.Lines {
		if showLineNumbers {
			lineNum := fmt.Sprintf("%3d │ ", line.LineNumber)
			result.WriteString(sh.theme.Delimiter.Render(lineNum))
		}

		for _, token := range line.Tokens {
			style := sh.getStyleForToken(token.Type, highlighted.Theme)
			result.WriteString(style.Render(token.Content))
		}

		result.WriteString("\n")
	}

	return result.String()
}

// getStyleForToken returns the appropriate style for a token type
func (sh *SyntaxHighlighter) getStyleForToken(tokenType TokenType, theme HighlightTheme) lipgloss.Style {
	switch tokenType {
	case TokenKeyword:
		return theme.Keyword
	case TokenString:
		return theme.String
	case TokenComment:
		return theme.Comment
	case TokenFunction:
		return theme.Function
	case TokenNumber:
		return theme.Number
	case TokenOperator:
		return theme.Operator
	case TokenType_:
		return theme.Type
	case TokenVariable:
		return theme.Variable
	case TokenBracket:
		return theme.Bracket
	case TokenDelimiter:
		return theme.Delimiter
	default:
		return lipgloss.NewStyle() // Plain text
	}
}

// GetSupportedLanguages returns a list of supported languages
func (sh *SyntaxHighlighter) GetSupportedLanguages() []string {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	var languages []string
	seen := make(map[string]bool)

	for lang := range sh.languages {
		if !seen[lang] {
			languages = append(languages, lang)
			seen[lang] = true
		}
	}

	return languages
}

// SetTheme updates the highlighting theme
func (sh *SyntaxHighlighter) SetTheme(theme HighlightTheme) {
	sh.mutex.Lock()
	sh.theme = theme
	// Clear cache since theme changed
	sh.cache = make(map[string]HighlightedCode)
	sh.mutex.Unlock()
}

// ClearCache clears the highlighting cache
func (sh *SyntaxHighlighter) ClearCache() {
	sh.mutex.Lock()
	sh.cache = make(map[string]HighlightedCode)
	sh.mutex.Unlock()
}
