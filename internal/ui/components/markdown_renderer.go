package components

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/common-creation/coda/internal/styles"
)

// MarkdownRenderer renders markdown content with terminal-friendly styling
type MarkdownRenderer struct {
	styles           styles.Styles
	highlighter      *SyntaxHighlighter
	maxWidth         int
	preserveNewlines bool
	listDepth        int
	quoteDepth       int
}

// MarkdownElement represents a parsed markdown element
type MarkdownElement struct {
	Type     ElementType
	Content  string
	Level    int
	Language string
	Items    []string
	Ordered  bool
}

// ElementType represents different markdown element types
type ElementType int

const (
	ElementParagraph ElementType = iota
	ElementHeading
	ElementCodeBlock
	ElementList
	ElementQuote
	ElementHorizontalRule
	ElementLineBreak
)

// NewMarkdownRenderer creates a new markdown renderer
func NewMarkdownRenderer(styles styles.Styles, highlighter *SyntaxHighlighter) *MarkdownRenderer {
	return &MarkdownRenderer{
		styles:           styles,
		highlighter:      highlighter,
		maxWidth:         80,
		preserveNewlines: false,
	}
}

// SetMaxWidth sets the maximum width for text wrapping
func (r *MarkdownRenderer) SetMaxWidth(width int) {
	r.maxWidth = width
}

// SetPreserveNewlines controls whether to preserve newlines
func (r *MarkdownRenderer) SetPreserveNewlines(preserve bool) {
	r.preserveNewlines = preserve
}

// Render renders markdown content to styled terminal output
func (r *MarkdownRenderer) Render(markdown string) string {
	if strings.TrimSpace(markdown) == "" {
		return ""
	}

	elements := r.parseMarkdown(markdown)
	return r.renderElements(elements)
}

// parseMarkdown parses markdown text into elements
func (r *MarkdownRenderer) parseMarkdown(markdown string) []MarkdownElement {
	var elements []MarkdownElement
	lines := strings.Split(markdown, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Skip empty lines (will be handled as line breaks)
		if strings.TrimSpace(line) == "" {
			if len(elements) > 0 && elements[len(elements)-1].Type != ElementLineBreak {
				elements = append(elements, MarkdownElement{Type: ElementLineBreak})
			}
			i++
			continue
		}

		// Check for horizontal rule
		if r.isHorizontalRule(line) {
			elements = append(elements, MarkdownElement{Type: ElementHorizontalRule})
			i++
			continue
		}

		// Check for headings
		if heading, level := r.parseHeading(line); heading != "" {
			elements = append(elements, MarkdownElement{
				Type:    ElementHeading,
				Content: heading,
				Level:   level,
			})
			i++
			continue
		}

		// Check for code blocks
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			codeBlock, language, consumed := r.parseCodeBlock(lines[i:])
			if codeBlock != "" {
				elements = append(elements, MarkdownElement{
					Type:     ElementCodeBlock,
					Content:  codeBlock,
					Language: language,
				})
				i += consumed
				continue
			}
		}

		// Check for lists
		if r.isList(line) {
			listItems, ordered, consumed := r.parseList(lines[i:])
			elements = append(elements, MarkdownElement{
				Type:    ElementList,
				Items:   listItems,
				Ordered: ordered,
			})
			i += consumed
			continue
		}

		// Check for quotes
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			quote, consumed := r.parseQuote(lines[i:])
			elements = append(elements, MarkdownElement{
				Type:    ElementQuote,
				Content: quote,
			})
			i += consumed
			continue
		}

		// Default to paragraph
		paragraph, consumed := r.parseParagraph(lines[i:])
		if paragraph != "" {
			elements = append(elements, MarkdownElement{
				Type:    ElementParagraph,
				Content: paragraph,
			})
		}
		i += consumed
	}

	return elements
}

// renderElements renders parsed elements to styled output
func (r *MarkdownRenderer) renderElements(elements []MarkdownElement) string {
	var result strings.Builder

	for i, element := range elements {
		rendered := r.renderElement(element)
		if rendered != "" {
			result.WriteString(rendered)

			// Add spacing between elements
			if i < len(elements)-1 && element.Type != ElementLineBreak {
				next := elements[i+1]
				if next.Type != ElementLineBreak {
					result.WriteString("\n")
				}
			}
		}
	}

	return result.String()
}

// renderElement renders a single markdown element
func (r *MarkdownRenderer) renderElement(element MarkdownElement) string {
	switch element.Type {
	case ElementHeading:
		return r.renderHeading(element.Content, element.Level)
	case ElementParagraph:
		return r.renderParagraph(element.Content)
	case ElementCodeBlock:
		return r.renderCodeBlock(element.Content, element.Language)
	case ElementList:
		return r.renderList(element.Items, element.Ordered)
	case ElementQuote:
		return r.renderQuote(element.Content)
	case ElementHorizontalRule:
		return r.renderHorizontalRule()
	case ElementLineBreak:
		return "\n"
	default:
		return element.Content
	}
}

// renderHeading renders a heading
func (r *MarkdownRenderer) renderHeading(content string, level int) string {
	content = r.renderInlineElements(content)

	var style lipgloss.Style
	var prefix string

	switch level {
	case 1:
		style = r.styles.Bold.Foreground(r.styles.Colors.Primary)
		prefix = "━━━ "
		content = prefix + content + " " + strings.Repeat("━", max(0, r.maxWidth-len(content)-len(prefix)-1))
	case 2:
		style = r.styles.Bold.Foreground(r.styles.Colors.Secondary)
		prefix = "── "
		content = prefix + content + " " + strings.Repeat("─", max(0, r.maxWidth-len(content)-len(prefix)-1))
	case 3:
		style = r.styles.Bold.Foreground(r.styles.Colors.Accent)
		content = "▶ " + content
	default:
		style = r.styles.Bold
		content = strings.Repeat("#", level) + " " + content
	}

	return style.Render(content) + "\n"
}

// renderParagraph renders a paragraph with inline elements
func (r *MarkdownRenderer) renderParagraph(content string) string {
	content = r.renderInlineElements(content)
	content = r.wrapText(content, r.maxWidth)
	return content + "\n"
}

// renderCodeBlock renders a code block with syntax highlighting
func (r *MarkdownRenderer) renderCodeBlock(content, language string) string {
	// Remove trailing newline if present
	content = strings.TrimSuffix(content, "\n")

	var rendered string
	if r.highlighter != nil && language != "" {
		highlighted := r.highlighter.Highlight(content, language)
		rendered = r.highlighter.Render(highlighted, true)
	} else {
		// Render as plain code
		lines := strings.Split(content, "\n")
		var result strings.Builder
		for i, line := range lines {
			lineNum := fmt.Sprintf("%3d │ ", i+1)
			result.WriteString(r.styles.Muted.Render(lineNum))
			result.WriteString(r.styles.Code.Render(line))
			result.WriteString("\n")
		}
		rendered = result.String()
	}

	// Add border
	border := r.styles.Border.
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(1).
		Width(r.maxWidth - 4)

	if language != "" {
		title := r.styles.Bold.Render("Code (" + language + ")")
		border = border.BorderTop(true).BorderTopForeground(r.styles.Colors.Border)
		rendered = title + "\n" + rendered
	}

	return border.Render(rendered) + "\n"
}

// renderList renders a list (ordered or unordered)
func (r *MarkdownRenderer) renderList(items []string, ordered bool) string {
	var result strings.Builder

	for i, item := range items {
		var bullet string
		if ordered {
			bullet = fmt.Sprintf("%d. ", i+1)
		} else {
			bullet = "• "
		}

		bulletStyle := r.styles.Bold.Foreground(r.styles.Colors.Primary)
		itemContent := r.renderInlineElements(item)
		itemContent = r.wrapText(itemContent, r.maxWidth-len(bullet))

		// Handle multi-line items
		lines := strings.Split(itemContent, "\n")
		for j, line := range lines {
			if j == 0 {
				result.WriteString(bulletStyle.Render(bullet) + line + "\n")
			} else {
				result.WriteString(strings.Repeat(" ", len(bullet)) + line + "\n")
			}
		}
	}

	return result.String()
}

// renderQuote renders a quote block
func (r *MarkdownRenderer) renderQuote(content string) string {
	content = r.renderInlineElements(content)
	lines := strings.Split(content, "\n")

	var result strings.Builder
	quoteStyle := r.styles.Quote
	borderStyle := r.styles.Muted.Foreground(r.styles.Colors.Border)

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			wrapped := r.wrapText(line, r.maxWidth-4)
			wrappedLines := strings.Split(wrapped, "\n")
			for _, wrappedLine := range wrappedLines {
				result.WriteString(borderStyle.Render("┃ "))
				result.WriteString(quoteStyle.Render(wrappedLine))
				result.WriteString("\n")
			}
		} else {
			result.WriteString(borderStyle.Render("┃"))
			result.WriteString("\n")
		}
	}

	return result.String()
}

// renderHorizontalRule renders a horizontal rule
func (r *MarkdownRenderer) renderHorizontalRule() string {
	rule := strings.Repeat("─", r.maxWidth)
	return r.styles.Muted.Render(rule) + "\n"
}

// renderInlineElements processes inline markdown elements
func (r *MarkdownRenderer) renderInlineElements(content string) string {
	// Bold text (**text** or __text__)
	boldRegex := regexp.MustCompile(`\*\*(.*?)\*\*|__(.*?)__`)
	content = boldRegex.ReplaceAllStringFunc(content, func(match string) string {
		text := strings.Trim(match, "*_")
		return r.styles.Bold.Render(text)
	})

	// Italic text (*text* or _text_)
	italicRegex := regexp.MustCompile(`(?:^|[^*])\*([^*]+)\*(?:[^*]|$)|(?:^|[^_])_([^_]+)_(?:[^_]|$)`)
	content = italicRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the content between * or _
		if strings.Contains(match, "*") {
			text := regexp.MustCompile(`\*([^*]+)\*`).FindStringSubmatch(match)
			if len(text) > 1 {
				prefix := strings.Split(match, "*"+text[1]+"*")[0]
				suffix := strings.Split(match, "*"+text[1]+"*")[1]
				return prefix + r.styles.Italic.Render(text[1]) + suffix
			}
		} else {
			text := regexp.MustCompile(`_([^_]+)_`).FindStringSubmatch(match)
			if len(text) > 1 {
				prefix := strings.Split(match, "_"+text[1]+"_")[0]
				suffix := strings.Split(match, "_"+text[1]+"_")[1]
				return prefix + r.styles.Italic.Render(text[1]) + suffix
			}
		}
		return match
	})

	// Inline code (`code`)
	codeRegex := regexp.MustCompile("`([^`]+)`")
	content = codeRegex.ReplaceAllStringFunc(content, func(match string) string {
		code := strings.Trim(match, "`")
		return r.styles.Code.Render(code)
	})

	// Links [text](url) - just show the text
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	content = linkRegex.ReplaceAllStringFunc(content, func(match string) string {
		matches := linkRegex.FindStringSubmatch(match)
		if len(matches) > 1 {
			return r.styles.Link.Render(matches[1])
		}
		return match
	})

	// Simple URLs
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	content = urlRegex.ReplaceAllStringFunc(content, func(url string) string {
		return r.styles.Link.Render(url)
	})

	return content
}

// wrapText wraps text to fit within the specified width
func (r *MarkdownRenderer) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var result strings.Builder
	var currentLine strings.Builder
	currentLength := 0

	for _, word := range words {
		// Calculate display width (ignoring ANSI codes)
		wordWidth := lipgloss.Width(word)

		if currentLength+wordWidth+1 > width && currentLength > 0 {
			// Start new line
			result.WriteString(currentLine.String())
			result.WriteString("\n")
			currentLine.Reset()
			currentLength = 0
		}

		if currentLength > 0 {
			currentLine.WriteString(" ")
			currentLength++
		}

		currentLine.WriteString(word)
		currentLength += wordWidth
	}

	if currentLine.Len() > 0 {
		result.WriteString(currentLine.String())
	}

	return result.String()
}

// Parsing helper methods

// parseHeading parses a heading and returns content and level
func (r *MarkdownRenderer) parseHeading(line string) (string, int) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return "", 0
	}

	level := 0
	for i, char := range trimmed {
		if char == '#' {
			level++
		} else if char == ' ' {
			return strings.TrimSpace(trimmed[i:]), level
		} else {
			break
		}
	}

	return "", 0
}

// parseCodeBlock parses a code block and returns content, language, and lines consumed
func (r *MarkdownRenderer) parseCodeBlock(lines []string) (string, string, int) {
	if len(lines) == 0 {
		return "", "", 0
	}

	firstLine := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(firstLine, "```") {
		return "", "", 0
	}

	language := strings.TrimPrefix(firstLine, "```")
	language = strings.TrimSpace(language)

	var content strings.Builder
	consumed := 1

	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "```" {
			consumed = i + 1
			break
		}
		content.WriteString(line)
		if i < len(lines)-1 {
			content.WriteString("\n")
		}
		consumed = i + 1
	}

	return content.String(), language, consumed
}

// isList checks if a line is a list item
func (r *MarkdownRenderer) isList(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	// Check for unordered list
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
		return true
	}

	// Check for ordered list
	if matched, _ := regexp.MatchString(`^\d+\.\s`, trimmed); matched {
		return true
	}

	return false
}

// parseList parses a list and returns items, ordered flag, and lines consumed
func (r *MarkdownRenderer) parseList(lines []string) ([]string, bool, int) {
	if len(lines) == 0 {
		return nil, false, 0
	}

	var items []string
	consumed := 0
	firstLine := strings.TrimSpace(lines[0])
	ordered := false

	// Determine if ordered
	if matched, _ := regexp.MatchString(`^\d+\.\s`, firstLine); matched {
		ordered = true
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			consumed++
			continue
		}

		if r.isList(line) {
			// Extract item content
			var content string
			if ordered {
				re := regexp.MustCompile(`^\d+\.\s(.*)`)
				matches := re.FindStringSubmatch(trimmed)
				if len(matches) > 1 {
					content = matches[1]
				}
			} else {
				if strings.HasPrefix(trimmed, "- ") {
					content = strings.TrimPrefix(trimmed, "- ")
				} else if strings.HasPrefix(trimmed, "* ") {
					content = strings.TrimPrefix(trimmed, "* ")
				} else if strings.HasPrefix(trimmed, "+ ") {
					content = strings.TrimPrefix(trimmed, "+ ")
				}
			}

			items = append(items, content)
			consumed++
		} else {
			break
		}
	}

	return items, ordered, consumed
}

// parseQuote parses a quote block and returns content and lines consumed
func (r *MarkdownRenderer) parseQuote(lines []string) (string, int) {
	var content strings.Builder
	consumed := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, ">") {
			quoteContent := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
			if content.Len() > 0 {
				content.WriteString("\n")
			}
			content.WriteString(quoteContent)
			consumed++
		} else if trimmed == "" {
			// Allow empty lines within quotes
			content.WriteString("\n")
			consumed++
		} else {
			break
		}
	}

	return content.String(), consumed
}

// parseParagraph parses a paragraph and returns content and lines consumed
func (r *MarkdownRenderer) parseParagraph(lines []string) (string, int) {
	var content strings.Builder
	consumed := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			break
		}

		// Stop at special markdown elements
		if r.isHorizontalRule(line) || r.isList(line) || strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "```") {
			break
		}

		if content.Len() > 0 {
			content.WriteString(" ")
		}
		content.WriteString(trimmed)
		consumed++
	}

	return content.String(), max(1, consumed)
}

// isHorizontalRule checks if a line is a horizontal rule
func (r *MarkdownRenderer) isHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}

	// Check for --- or *** or ___
	return strings.Count(trimmed, "-") >= 3 && strings.ReplaceAll(trimmed, "-", "") == "" ||
		strings.Count(trimmed, "*") >= 3 && strings.ReplaceAll(trimmed, "*", "") == "" ||
		strings.Count(trimmed, "_") >= 3 && strings.ReplaceAll(trimmed, "_", "") == ""
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
