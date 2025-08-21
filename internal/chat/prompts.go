package chat

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"
)

// PromptBuilder manages system prompt construction
type PromptBuilder struct {
	basePrompt    string
	toolPrompts   map[string]string
	contextInfo   ContextInfo
	templates     map[string]*template.Template
	maxTokens     int
	tokenCounter  TokenCounter
	customPrompts map[string]string
}

// ContextInfo contains contextual information for prompt building
type ContextInfo struct {
	WorkingDir   string            `json:"working_dir"`
	Platform     string            `json:"platform"`
	UserName     string            `json:"user_name"`
	ProjectInfo  map[string]string `json:"project_info"`
	Timestamp    time.Time         `json:"timestamp"`
	SessionID    string            `json:"session_id"`
	Language     string            `json:"language"`
	CustomFields map[string]string `json:"custom_fields"`
}

// PromptTemplate represents a reusable prompt template
type PromptTemplate struct {
	Name        string
	Template    string
	Priority    int
	MaxTokens   int
	RequiredCtx []string
}

// DefaultPromptTemplates provides standard prompt templates
var DefaultPromptTemplates = map[string]PromptTemplate{
	"base": {
		Name: "base",
		Template: `You are CODA (CODing Agent), an advanced AI coding assistant specialized in helping developers with software engineering tasks.
{{if .WorkingDir}}Current directory: {{.WorkingDir}}{{end}}
{{if .Platform}}Platform: {{.Platform}}{{end}}
{{if .Timestamp}}Current time: {{.Timestamp.Format "2006-01-02 15:04:05"}}{{end}}

## YOUR PRIMARY DIRECTIVE
**YOU MUST ACTIVELY USE TOOLS TO INTERACT WITH FILES AND CODE.** When users ask about files, code, or project structure, immediately use the appropriate tools to gather information. Never ask users for file paths or content - find and read them yourself using tools.

## Core Capabilities
- **Code Analysis**: Use tools to read and analyze code structure, dependencies, and complexity
- **Documentation Generation**: Read files with tools and create comprehensive documentation
- **Code Understanding**: Use tools to explore and explain complex code flows
- **Modernization**: Read code with tools to suggest refactoring strategies
- **Interactive Assistance**: Proactively use tools to answer questions about codebase`,
		Priority: 100,
	},
	"tools": {
		Name: "tools",
		Template: `
## Available Tools
You have access to various tools for file operations and code analysis:
{{range .Tools}}
- **{{.Name}}**: {{.Description}}{{end}}

## CRITICAL TOOL USAGE RULES
**YOU MUST ALWAYS USE TOOLS TO INTERACT WITH FILES.** Never ask the user for file paths or content - use tools to find and read them yourself.

### MANDATORY Tool Usage Scenarios:
1. **When asked to read ANY file** → IMMEDIATELY use "read_file" tool (e.g., "README.md", "package.json", etc.)
2. **When text starts with @** → Treat it as a file path and use "read_file" tool (e.g., "@README.md", "@src/main.go")
3. **When asked to summarize content** → FIRST use "read_file" to get the content, THEN summarize
4. **When you need to find files** → Use "list_files" or "search_files" tools
5. **When asked about project structure** → Use "list_files" to explore directories
6. **When modifying files** → Use "write_file" or "edit_file" tools

### Response Format:
When using tools, respond with ONLY the structured JSON (no additional text):
{
  "response_type": "tool_call",
  "text": null,
  "tool_calls": [
    {
      "tool": "read_file",
      "arguments": {"path": "README.md"}
    }
  ]
}

### Examples:
- User: "README.mdを要約してください" 
  → Use: {"tool": "read_file", "arguments": {"path": "README.md"}}
  
- User: "@README.md を要約して"
  → Use: {"tool": "read_file", "arguments": {"path": "README.md"}}

- User: "What files are in the src directory?"
  → Use: {"tool": "list_files", "arguments": {"path": "src"}}
  
- User: "Find all test files"
  → Use: {"tool": "search_files", "arguments": {"pattern": "test", "filePattern": "*.test.*"}}

**REMEMBER**: ALWAYS try to use tools FIRST before asking for clarification. If a file doesn't exist, the tool will tell you.`,
		Priority: 95,
	},
	"project": {
		Name: "project",
		Template: `{{if .ProjectInfo}}
Project Information:
{{range $key, $value := .ProjectInfo}}- {{$key}}: {{$value}}
{{end}}{{end}}`,
		Priority: 80,
	},
	"language": {
		Name: "language",
		Template: `Please respond in English. Use clear and concise language.

## Language Response Guidelines
Detect the user's language from their input and respond in the same language:
- "Hello" → Respond in English
- "こんにちは" → Respond in Japanese  
- "안녕하세요" → Respond in Korean
- Default to English if language is unclear

## Important Instructions
When asked about files, **ALWAYS use tools to read them first.**
- Example: "Summarize README.md" → Immediately use read_file tool
- **Text starting with @ indicates a file path** → Example: "@README.md" "@src/main.go" → Immediately use read_file tool`,
		Priority: 70,
	},
	"best_practices": {
		Name: "best_practices",
		Template: `
## Workflow Guidelines
1. **Tool-First Approach**: ALWAYS use tools to interact with files - never assume or guess
2. **Understanding Phase**: Use tools to explore and understand the codebase structure
3. **Analysis Phase**: Read files directly with tools to analyze code
4. **Documentation Phase**: Generate documentation based on actual file content
5. **Verification Phase**: Use tools to verify changes and file states

## Best Practices
- **ALWAYS USE TOOLS FIRST** - Don't ask for file locations, search for them
- **Be Proactive** - When asked about a file, immediately read it with tools
- **Explore Automatically** - Use list_files to understand project structure
- **Search Don't Guess** - Use search_files to find relevant code
- Preserve original code formatting and style
- Generate documentation that follows language-specific conventions
- Provide clear explanations for complex logic
- Be security-conscious and never expose sensitive information

## Tool Usage Priority
1. **HIGHEST**: read_file - Use this immediately when any file is mentioned
2. **HIGH**: list_files - Use to explore directories and find files
3. **HIGH**: search_files - Use to find specific patterns or files
4. **MEDIUM**: edit_file - Use to modify existing files
5. **LOW**: write_file - Use only when creating new files

## Response Format
- When using tools, return structured JSON immediately
- After tool results, provide Markdown-formatted analysis
- Include code blocks with appropriate syntax highlighting
- Structure responses with clear sections and bullet points
- When showing file paths, use the format: ` + "`" + `path/to/file.ext:line_number` + "`" + `

Remember: Your primary goal is to help developers by ACTIVELY using tools to work with their code.`,
		Priority: 85,
	},
}

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder(maxTokens int, tokenCounter TokenCounter) *PromptBuilder {
	if tokenCounter == nil {
		tokenCounter = &SimpleTokenCounter{}
	}

	pb := &PromptBuilder{
		basePrompt:    DefaultPromptTemplates["base"].Template,
		toolPrompts:   make(map[string]string),
		templates:     make(map[string]*template.Template),
		maxTokens:     maxTokens,
		tokenCounter:  tokenCounter,
		customPrompts: make(map[string]string),
	}

	// Initialize default templates
	for name, tmpl := range DefaultPromptTemplates {
		t, err := template.New(name).Parse(tmpl.Template)
		if err == nil {
			pb.templates[name] = t
		}
	}

	// Initialize context info
	pb.contextInfo = pb.gatherContextInfo()

	return pb
}

// gatherContextInfo collects system and environment information
func (pb *PromptBuilder) gatherContextInfo() ContextInfo {
	ctx := ContextInfo{
		Platform:     runtime.GOOS,
		Timestamp:    time.Now(),
		ProjectInfo:  make(map[string]string),
		CustomFields: make(map[string]string),
	}

	// Get working directory
	if wd, err := os.Getwd(); err == nil {
		ctx.WorkingDir = wd
	}

	// Get username
	if user := os.Getenv("USER"); user != "" {
		ctx.UserName = user
	} else if user := os.Getenv("USERNAME"); user != "" {
		ctx.UserName = user
	}

	// Detect project type
	pb.detectProjectInfo(&ctx)

	// Get language preference
	if lang := os.Getenv("LANG"); lang != "" {
		if strings.HasPrefix(lang, "ja") {
			ctx.Language = "ja"
		} else if strings.HasPrefix(lang, "zh") {
			ctx.Language = "zh"
		} else {
			ctx.Language = "en"
		}
	} else {
		ctx.Language = "en"
	}

	return ctx
}

// detectProjectInfo detects project type and information
func (pb *PromptBuilder) detectProjectInfo(ctx *ContextInfo) {
	if ctx.WorkingDir == "" {
		return
	}

	// Check for Go project
	if _, err := os.Stat(filepath.Join(ctx.WorkingDir, "go.mod")); err == nil {
		ctx.ProjectInfo["type"] = "Go"
		if content, err := os.ReadFile(filepath.Join(ctx.WorkingDir, "go.mod")); err == nil {
			lines := strings.Split(string(content), "\n")
			if len(lines) > 0 && strings.HasPrefix(lines[0], "module ") {
				ctx.ProjectInfo["module"] = strings.TrimPrefix(lines[0], "module ")
			}
		}
	}

	// Check for Node.js project
	if _, err := os.Stat(filepath.Join(ctx.WorkingDir, "package.json")); err == nil {
		ctx.ProjectInfo["type"] = "Node.js"
	}

	// Check for Python project
	if _, err := os.Stat(filepath.Join(ctx.WorkingDir, "requirements.txt")); err == nil {
		ctx.ProjectInfo["type"] = "Python"
	} else if _, err := os.Stat(filepath.Join(ctx.WorkingDir, "pyproject.toml")); err == nil {
		ctx.ProjectInfo["type"] = "Python"
	}

	// Check for Git repository
	if _, err := os.Stat(filepath.Join(ctx.WorkingDir, ".git")); err == nil {
		ctx.ProjectInfo["vcs"] = "git"
	}
}

// Build constructs the complete system prompt
func (pb *PromptBuilder) Build() (string, error) {
	var parts []PromptPart

	// Gather all prompt parts
	for name, tmpl := range pb.templates {
		part, err := pb.renderTemplate(name, tmpl)
		if err != nil {
			continue
		}

		if part != "" {
			priority := DefaultPromptTemplates[name].Priority
			if priority == 0 {
				priority = 50 // Default priority
			}

			parts = append(parts, PromptPart{
				Name:     name,
				Content:  part,
				Priority: priority,
				Tokens:   pb.tokenCounter.CountTokens(part),
			})
		}
	}

	// Add custom prompts
	for name, content := range pb.customPrompts {
		parts = append(parts, PromptPart{
			Name:     name,
			Content:  content,
			Priority: 60, // Default priority for custom prompts
			Tokens:   pb.tokenCounter.CountTokens(content),
		})
	}

	// Sort by priority and optimize for token limit
	optimized := pb.optimizePromptParts(parts)

	// Combine parts
	var result strings.Builder
	for _, part := range optimized {
		if result.Len() > 0 {
			result.WriteString("\n\n")
		}
		result.WriteString(strings.TrimSpace(part.Content))
	}

	return result.String(), nil
}

// renderTemplate renders a template with context
func (pb *PromptBuilder) renderTemplate(name string, tmpl *template.Template) (string, error) {
	data := struct {
		ContextInfo
		Tools []ToolInfo
	}{
		ContextInfo: pb.contextInfo,
		Tools:       pb.getToolInfo(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return strings.TrimSpace(buf.String()), nil
}

// getToolInfo returns information about available tools
func (pb *PromptBuilder) getToolInfo() []ToolInfo {
	var tools []ToolInfo
	for name, desc := range pb.toolPrompts {
		tools = append(tools, ToolInfo{
			Name:        name,
			Description: desc,
		})
	}
	return tools
}

// ToolInfo represents tool information for templates
type ToolInfo struct {
	Name        string
	Description string
}

// PromptPart represents a part of the prompt with metadata
type PromptPart struct {
	Name     string
	Content  string
	Priority int
	Tokens   int
}

// optimizePromptParts optimizes prompt parts to fit within token limit
func (pb *PromptBuilder) optimizePromptParts(parts []PromptPart) []PromptPart {
	// Sort by priority (higher first)
	sortedParts := make([]PromptPart, len(parts))
	copy(sortedParts, parts)

	for i := 0; i < len(sortedParts)-1; i++ {
		for j := i + 1; j < len(sortedParts); j++ {
			if sortedParts[i].Priority < sortedParts[j].Priority {
				sortedParts[i], sortedParts[j] = sortedParts[j], sortedParts[i]
			}
		}
	}

	// Select parts that fit within token limit
	var selected []PromptPart
	totalTokens := 0

	for _, part := range sortedParts {
		if totalTokens+part.Tokens <= pb.maxTokens {
			selected = append(selected, part)
			totalTokens += part.Tokens
		}
	}

	return selected
}

// SetContextField sets a custom context field
func (pb *PromptBuilder) SetContextField(key, value string) {
	pb.contextInfo.CustomFields[key] = value
}

// SetProjectInfo sets project information
func (pb *PromptBuilder) SetProjectInfo(key, value string) {
	pb.contextInfo.ProjectInfo[key] = value
}

// SetSessionID sets the current session ID
func (pb *PromptBuilder) SetSessionID(sessionID string) {
	pb.contextInfo.SessionID = sessionID
}

// AddToolPrompt adds or updates a tool prompt
func (pb *PromptBuilder) AddToolPrompt(name, description string) {
	pb.toolPrompts[name] = description
}

// RemoveToolPrompt removes a tool prompt
func (pb *PromptBuilder) RemoveToolPrompt(name string) {
	delete(pb.toolPrompts, name)
}

// AddCustomPrompt adds a custom prompt section
func (pb *PromptBuilder) AddCustomPrompt(name, content string) {
	pb.customPrompts[name] = content
}

// RemoveCustomPrompt removes a custom prompt section
func (pb *PromptBuilder) RemoveCustomPrompt(name string) {
	delete(pb.customPrompts, name)
}

// AddTemplate adds a new template
func (pb *PromptBuilder) AddTemplate(name, templateStr string, priority int) error {
	tmpl, err := template.New(name).Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	pb.templates[name] = tmpl

	// Update default templates if it exists
	if _, exists := DefaultPromptTemplates[name]; exists {
		t := DefaultPromptTemplates[name]
		t.Template = templateStr
		t.Priority = priority
		DefaultPromptTemplates[name] = t
	}

	return nil
}

// UpdateContext refreshes the context information
func (pb *PromptBuilder) UpdateContext() {
	pb.contextInfo = pb.gatherContextInfo()
}

// GetTokenCount returns the token count for the current prompt
func (pb *PromptBuilder) GetTokenCount() (int, error) {
	prompt, err := pb.Build()
	if err != nil {
		return 0, err
	}
	return pb.tokenCounter.CountTokens(prompt), nil
}

// Clone creates a copy of the prompt builder
func (pb *PromptBuilder) Clone() *PromptBuilder {
	newPB := &PromptBuilder{
		basePrompt:    pb.basePrompt,
		toolPrompts:   make(map[string]string),
		templates:     make(map[string]*template.Template),
		maxTokens:     pb.maxTokens,
		tokenCounter:  pb.tokenCounter,
		customPrompts: make(map[string]string),
		contextInfo:   pb.contextInfo,
	}

	// Copy tool prompts
	for k, v := range pb.toolPrompts {
		newPB.toolPrompts[k] = v
	}

	// Copy templates
	for k, v := range pb.templates {
		newPB.templates[k] = v
	}

	// Copy custom prompts
	for k, v := range pb.customPrompts {
		newPB.customPrompts[k] = v
	}

	return newPB
}
