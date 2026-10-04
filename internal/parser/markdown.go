package parser

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

var defaultMarkdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.TaskList,
	),
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
	),
)

var checkboxRe = regexp.MustCompile(`<input\s+([^>]*?)type="checkbox"([^>]*?)/?>`)

// AcceptanceCriterion represents an individual checkbox item extracted from task markdown.
type AcceptanceCriterion struct {
	Index     int    `json:"index"`
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

// Heading represents an extracted markdown heading with its nesting level.
type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

// ExtractAcceptanceCriteria parses markdown content using Goldmark AST inspection
// and returns all task checklist criteria along with counts.
func ExtractAcceptanceCriteria(source []byte) (total int, completed int, items []AcceptanceCriterion) {
	if len(source) == 0 {
		return 0, 0, nil
	}

	reader := text.NewReader(source)
	doc := defaultMarkdown.Parser().Parse(reader)

	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}

		if n.Kind() == extast.KindTaskCheckBox {
			cb := n.(*extast.TaskCheckBox)
			isCompleted := cb.IsChecked
			total++

			// Collect the textual label of this checkbox item
			var label strings.Builder
			for sibling := cb.NextSibling(); sibling != nil; sibling = sibling.NextSibling() {
				collectNodeText(sibling, source, &label)
			}

			criterion := AcceptanceCriterion{
				Index:     total,
				Text:      strings.TrimSpace(label.String()),
				Completed: isCompleted,
			}

			items = append(items, criterion)
			if isCompleted {
				completed++
			}
		}

		return gast.WalkContinue, nil
	})

	return total, completed, items
}

// HasOpenCheckboxes returns true if any checkbox in the markdown source is uncompleted.
func HasOpenCheckboxes(source []byte) bool {
	total, completed, _ := ExtractAcceptanceCriteria(source)
	return total > completed
}

// ExtractHeadings extracts all markdown headings and their levels.
func ExtractHeadings(source []byte) []Heading {
	if len(source) == 0 {
		return nil
	}

	reader := text.NewReader(source)
	doc := defaultMarkdown.Parser().Parse(reader)

	var headings []Heading

	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}

		if n.Kind() == gast.KindHeading {
			h := n.(*gast.Heading)
			var headingText strings.Builder
			for child := h.FirstChild(); child != nil; child = child.NextSibling() {
				collectNodeText(child, source, &headingText)
			}

			headings = append(headings, Heading{
				Level: h.Level,
				Text:  strings.TrimSpace(headingText.String()),
			})
		}

		return gast.WalkContinue, nil
	})

	return headings
}

// RenderHTML converts markdown source bytes into HTML using Goldmark,
// styling task list checkboxes with daisyUI classes and 1-based data-checkbox-index attributes.
func RenderHTML(source []byte) (string, error) {
	if len(source) == 0 {
		return "", nil
	}

	var buf bytes.Buffer
	if err := defaultMarkdown.Convert(source, &buf); err != nil {
		return "", err
	}

	rawHTML := buf.String()
	cbIndex := 0
	formattedHTML := checkboxRe.ReplaceAllStringFunc(rawHTML, func(match string) string {
		cbIndex++
		checked := strings.Contains(match, "checked")
		checkedAttr := ""
		if checked {
			checkedAttr = " checked"
		}
		// name keeps browsers' form-field audits happy (fields need an id or name).
		return fmt.Sprintf(`<input type="checkbox"%s name="criterion-%d" class="checkbox checkbox-primary checkbox-xs mt-0.5 shrink-0 cursor-pointer" data-checkbox-index="%d" />`, checkedAttr, cbIndex, cbIndex)
	})

	return formattedHTML, nil
}

// ParseTaskWithCriteria parses a complete task document and calculates
// TotalCriteria and CompletedCriteria metrics from its markdown body.
func ParseTaskWithCriteria(content []byte, id string) (*model.Task, error) {
	task, err := ParseTask(content, id)
	if err != nil {
		return nil, err
	}

	total, completed, _ := ExtractAcceptanceCriteria([]byte(task.Body))
	task.TotalCriteria = total
	task.CompletedCriteria = completed

	return task, nil
}

// collectNodeText recursively extracts raw text from an AST node.
func collectNodeText(n gast.Node, source []byte, buf *strings.Builder) {
	switch node := n.(type) {
	case *gast.Text:
		buf.Write(node.Segment.Value(source))
	case *gast.String:
		buf.Write(node.Value)
	default:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			collectNodeText(child, source, buf)
		}
	}
}
