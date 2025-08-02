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
		Template: `You are CODA (CODing Agent), an AI assistant designed to help developers with coding tasks.
{{if .WorkingDir}}Current directory: {{.WorkingDir}}{{end}}
{{if .Platform}}Platform: {{.Platform}}{{end}}
{{if .Timestamp}}Current time: {{.Timestamp.Format "2006-01-02 15:04:05"}}{{end}}

You have access to various tools for file operations and can execute them to assist with programming tasks.
Always be helpful, accurate, and provide clear explanations for your actions.

When users mention paths starting with @ in their messages, you can read, write, or search these files by using the appropriate tools with those paths.`,
		Priority: 100,
	},
	"tools": {
		Name: "tools",
		Template: `
Available tools:
{{range .Tools}}
- {{.Name}}: {{.Description}}{{end}}

When using tools:
1. Always explain what you're about to do before using a tool
2. Request user approval for potentially dangerous operations
3. Handle errors gracefully and suggest alternatives
4. Validate inputs before executing tools`,
		Priority: 90,
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
		Template: `{{if eq .Language "ja"}}
日本語で応答してください。技術用語は適切に使用し、必要に応じて英語の用語も併記してください。
{{else if eq .Language "zh"}}
请用中文回复。适当使用技术术语，必要时可以附加英文术语。
{{else}}
Please respond in English. Use clear and concise language.
{{end}}`,
		Priority: 70,
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
