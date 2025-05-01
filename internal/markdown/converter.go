package markdown

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/yourusername/yourproject/links"
)

// Converter converts HTML content to Markdown
type Converter struct {
	BaseURL     string
	LinkManager *links.LinkManager
}

// NewConverter creates a new HTML to Markdown converter
func NewConverter(baseURL string, linkManager *links.LinkManager) *Converter {
	return &Converter{
		BaseURL:     baseURL,
		LinkManager: linkManager,
	}
}

// HTMLToMarkdown converts HTML content to Markdown format
func (c *Converter) HTMLToMarkdown(title, html string, metadata map[string]string) (string, error) {
	// Load HTML into goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("error parsing HTML: %w", err)
	}

	// Clean the HTML
	c.cleanHTML(doc)

	var sb strings.Builder

	// Create YAML frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: \"%s\"\n", escapeYAML(title)))
	sb.WriteString(fmt.Sprintf("source: \"%s\"\n", c.BaseURL))
	sb.WriteString(fmt.Sprintf("date_scraped: \"%s\"\n", time.Now().Format("2006-01-02")))

	// Add any useful metadata
	for key, value := range metadata {
		switch key {
		case "description", "author", "keywords", "og:description", "article:published_time", "article:modified_time":
			// Clean up the key name for OG metadata
			cleanKey := strings.Replace(key, "og:", "", 1)
			cleanKey = strings.Replace(cleanKey, "article:", "", 1)
			sb.WriteString(fmt.Sprintf("%s: \"%s\"\n", cleanKey, escapeYAML(value)))
		}
	}
	sb.WriteString("---\n\n")

	// Add title as H1
	sb.WriteString("# " + title + "\n\n")

	// Process the HTML elements
	doc.Find("body").Children().Each(func(i int, s *goquery.Selection) {
		c.processElement(&sb, s, 0)
	})

	// Clean up extra newlines
	content := sb.String()
	content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")

	return content, nil
}

// Clean up the HTML before processing
func (c *Converter) cleanHTML(doc *goquery.Document) {
	// Remove script and style elements
	doc.Find("script, style, noscript, iframe, form").Remove()

	// Remove comments
	doc.Find("*").Contents().Each(func(i int, s *goquery.Selection) {
		if s.Nodes != nil && len(s.Nodes) > 0 && s.Nodes[0].Type == 8 { // Comment node
			s.Remove()
		}
	})
}

// Process an HTML element and convert it to Markdown
func (c *Converter) processElement(sb *strings.Builder, s *goquery.Selection, depth int) {
	if s.Is("h1, h2, h3, h4, h5, h6") {
		// Handle headings
		level := 0
		switch s.Get(0).Data {
		case "h1":
			level = 1
		case "h2":
			level = 2
		case "h3":
			level = 3
		case "h4":
			level = 4
		case "h5":
			level = 5
		case "h6":
			level = 6
		}
		sb.WriteString("\n" + strings.Repeat("#", level) + " " + s.Text() + "\n\n")
	} else if s.Is("p") {
		// Handle paragraphs
		text := strings.TrimSpace(s.Text())
		if text != "" {
			sb.WriteString(text + "\n\n")
		}
	} else if s.Is("ul, ol") {
		// Handle lists
		sb.WriteString("\n")
		s.Find("li").Each(func(i int, li *goquery.Selection) {
			if s.Is("ul") {
				sb.WriteString("- " + strings.TrimSpace(li.Text()) + "\n")
			} else {
				sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, strings.TrimSpace(li.Text())))
			}
		})
		sb.WriteString("\n")
	} else if s.Is("blockquote") {
		// Handle blockquotes
		sb.WriteString("> " + strings.TrimSpace(s.Text()) + "\n\n")
	} else if s.Is("a") {
		// Handle links
		href, exists := s.Attr("href")
		text := s.Text()
		if exists && text != "" {
			// Try to convert to Obsidian internal link format
			if c.LinkManager != nil {
				obsidianLink, isInternal := c.LinkManager.GetObsidianLink(href, text)
				if isInternal {
					sb.WriteString(obsidianLink)
				} else {
					// Use regular markdown link for external links
					sb.WriteString("[" + text + "](" + href + ")")
				}
			} else {
				// Fallback to regular markdown links if no link manager
				sb.WriteString("[" + text + "](" + href + ")")
			}
		}
	} else if s.Is("img") {
		// Handle images
		alt, _ := s.Attr("alt")
		src, exists := s.Attr("src")
		if exists {
			// Make relative URLs absolute
			if !strings.HasPrefix(src, "http") {
				if strings.HasPrefix(src, "/") {
					src = c.BaseURL + src
				} else {
					src = c.BaseURL + "/" + src
				}
			}
			sb.WriteString("![" + alt + "](" + src + ")\n\n")
		}
	} else if s.Is("pre") {
		// Handle code blocks
		code := s.Find("code")
		language := ""
		class, exists := code.Attr("class")
		if exists && strings.Contains(class, "language-") {
			language = strings.Split(class, "language-")[1]
		}

		sb.WriteString("\n```" + language + "\n")
		sb.WriteString(strings.TrimSpace(code.Text()))
		sb.WriteString("\n```\n\n")
	} else if s.Is("code") {
		// Handle inline code
		sb.WriteString("`" + s.Text() + "`")
	} else if s.Is("hr") {
		// Handle horizontal rules
		sb.WriteString("\n---\n\n")
	} else if s.Is("br") {
		sb.WriteString("\n")
	} else if s.Is("table") {
		// Handle tables
		c.processTable(sb, s)
	} else {
		// Process child elements recursively
		s.Contents().Each(func(i int, child *goquery.Selection) {
			if child.Nodes != nil && len(child.Nodes) > 0 {
				if child.Nodes[0].Type == 1 { // Element node
					c.processElement(sb, child, depth+1)
				} else if child.Nodes[0].Type == 3 { // Text node
					text := strings.TrimSpace(child.Text())
					if text != "" {
						sb.WriteString(text)
					}
				}
			}
		})
	}
}

// Process HTML tables and convert to Markdown tables
func (c *Converter) processTable(sb *strings.Builder, table *goquery.Selection) {
	sb.WriteString("\n")

	// Process table headers
	headers := []string{}
	table.Find("thead tr th").Each(func(i int, s *goquery.Selection) {
		headers = append(headers, strings.TrimSpace(s.Text()))
	})

	// If no headers found in thead, check first tr
	if len(headers) == 0 {
		table.Find("tr:first-child th, tr:first-child td").Each(func(i int, s *goquery.Selection) {
			headers = append(headers, strings.TrimSpace(s.Text()))
		})
	}

	// Write headers
	for i, header := range headers {
		if i > 0 {
			sb.WriteString(" | ")
		}
		sb.WriteString(header)
	}
	sb.WriteString("\n")

	// Write separator row
	for i := range headers {
		if i > 0 {
			sb.WriteString(" | ")
		}
		sb.WriteString("---")
	}
	sb.WriteString("\n")

	// Write data rows
	rowSelector := "tbody tr"
	if table.Find("thead").Length() == 0 {
		// If no thead, skip the first row as it's the header
		rowSelector = "tr:not(:first-child)"
	}

	table.Find(rowSelector).Each(func(i int, row *goquery.Selection) {
		row.Find("td").Each(func(j int, cell *goquery.Selection) {
			if j > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString(strings.TrimSpace(cell.Text()))
		})
		sb.WriteString("\n")
	})

	sb.WriteString("\n")
}

// escapeYAML escapes special characters in YAML strings
func escapeYAML(s string) string {
	s = strings.Replace(s, "\"", "\\\"", -1)
	return s
}