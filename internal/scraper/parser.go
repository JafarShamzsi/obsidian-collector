package scraper

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ParsedContent represents the parsed content from a web page
type ParsedContent struct {
	Title    string
	Content  string
	Metadata map[string]string
}

// Parse extracts relevant content from the HTML document
func Parse(doc *goquery.Document, selector string) (*ParsedContent, error) {
	parsed := &ParsedContent{
		Metadata: make(map[string]string),
	}

	// Extract the page title
	parsed.Title = strings.TrimSpace(doc.Find("title").Text())

	// Extract metadata
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		property, _ := s.Attr("property")
		content, _ := s.Attr("content")

		// Store standard metadata and Open Graph metadata
		if name != "" && content != "" {
			parsed.Metadata[name] = content
		} else if property != "" && content != "" && strings.HasPrefix(property, "og:") {
			parsed.Metadata[property] = content
		}
	})

	// If a specific selector is provided, use it to extract content
	var contentBuilder strings.Builder
	if selector != "" {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			html, _ := s.Html()
			contentBuilder.WriteString(html)
			contentBuilder.WriteString("\n")
		})
	} else {
		// Default content extraction strategy - get main content area
		// Try common selectors for main content
		mainSelectors := []string{"main", "article", "#content", ".content", ".post", ".entry"}

		for _, sel := range mainSelectors {
			content := doc.Find(sel)
			if content.Length() > 0 {
				html, _ := content.Html()
				contentBuilder.WriteString(html)
				break
			}
		}

		// If no main content found, use body as fallback
		if contentBuilder.Len() == 0 {
			body := doc.Find("body")
			html, _ := body.Html()
			contentBuilder.WriteString(html)
		}
	}

	parsed.Content = contentBuilder.String()
	return parsed, nil
}