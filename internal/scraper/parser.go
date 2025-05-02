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
		mainSelectors := []string{
			"article", "main", ".post-content", ".article-content", ".entry-content", 
			"#content", ".content", ".post", ".article", ".entry", 
			"[itemprop='articleBody']", ".story", ".story-body",
		}
		
		var mainContent *goquery.Selection
		
		for _, sel := range mainSelectors {
			content := doc.Find(sel)
			if content.Length() > 0 && len(strings.TrimSpace(content.Text())) > 200 {
				html, _ := content.Html()
				contentBuilder.WriteString(html)
				mainContent = content
				break
			}
		}
		
		// If no main content found with known selectors, 
		// try to find the densest content area
		if mainContent == nil {
			bestElement := findDensestElement(doc)
			if bestElement != nil {
				html, _ := bestElement.Html()
				contentBuilder.WriteString(html)
			} else {
				// Last resort: use body as fallback
				body := doc.Find("body")
				html, _ := body.Html()
				contentBuilder.WriteString(html)
			}
		}
	}

	parsed.Content = contentBuilder.String()
	return parsed, nil
}

// findDensestElement finds the element with the highest text-to-HTML ratio
// that likely contains the main content
func findDensestElement(doc *goquery.Document) *goquery.Selection {
	var bestElement *goquery.Selection
	bestRatio := 0.0
	minTextLength := 200 // Minimum text length to consider

	// Find all potential content containers
	containers := doc.Find("div, section, article").FilterFunction(func(i int, s *goquery.Selection) bool {
		// Must contain paragraphs and have sufficient text
		return s.Find("p").Length() >= 2 && len(s.Text()) >= minTextLength
	})

	containers.Each(func(i int, s *goquery.Selection) {
		html, _ := s.Html()
		text := strings.TrimSpace(s.Text())

		textLength := len(text)
		htmlLength := len(html)

		if htmlLength > 0 {
			ratio := float64(textLength) / float64(htmlLength)

			// Boost ratio for elements with headings and fewer links
			headingBoost := float64(s.Find("h1, h2, h3, h4, h5, h6").Length()) * 0.1
			linkPenalty := float64(s.Find("a").Length()) * 0.02

			adjustedRatio := ratio + headingBoost - linkPenalty

			// Additional boost for longer content
			if textLength > 1000 {
				adjustedRatio += 0.2
			}

			// If this element has a better ratio, select it as best
			if adjustedRatio > bestRatio && textLength >= minTextLength {
				bestRatio = adjustedRatio
				bestElement = s
			}
		}
	})

	return bestElement
}