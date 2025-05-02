package markdown

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	  // Add this import
	"obsidian-collector/internal/links"
	"obsidian-collector/internal/config"
)

// Converter converts HTML content to Markdown
type Converter struct {
	BaseURL       string
	LinkManager   *links.LinkManager
	CleaningOptions struct {
		RemoveAds          bool
		RemoveNavigation   bool
		RemoveFooters      bool
		ExtractMainContent bool
	}
}

// NewConverter creates a new HTML to Markdown converter
func NewConverter(baseURL string, linkManager *links.LinkManager, config *config.Config) *Converter {
    c := &Converter{
        BaseURL:     baseURL,
        LinkManager: linkManager,
    }
    
    // Apply cleaning options from config
    c.CleaningOptions.RemoveAds = config.RemoveAds
    c.CleaningOptions.RemoveNavigation = config.RemoveNavigation
    c.CleaningOptions.RemoveFooters = config.RemoveFooters
    c.CleaningOptions.ExtractMainContent = config.ExtractMainContent
    
    return c
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
    
    // Remove common ad containers, sidebars, navbars, footers
    adSelectors := []string{
        "div[class*='ads']", "div[class*='advertisement']", "div[id*='ads']",
        "div[class*='banner']", "div[id*='banner']", 
        "aside", ".sidebar", "#sidebar", "[class*='sidebar']",
        "footer", ".footer", "#footer", "[class*='footer']",
        "nav", ".nav", "#nav", ".navigation", "#navigation",
        "[class*='cookie']", "[id*='cookie']",
        ".social", "#social", "[class*='social']",
        "[class*='related']", "[id*='related']",
        "[class*='share']", "[id*='share']",
        "[class*='newsletter']", "[id*='newsletter']",
        "[class*='popup']", "[id*='popup']",
        "[class*='advert']", "[id*='advert']",
        ".comments", "#comments", "[class*='comment-']",
        "header", ".header", "#header",
    }
    
    doc.Find(strings.Join(adSelectors, ", ")).Remove()
    
    // Remove comments
    doc.Find("*").Contents().Each(func(i int, s *goquery.Selection) {
        if s.Nodes != nil && len(s.Nodes) > 0 && s.Nodes[0].Type == 8 { // Comment node
            s.Remove()
        }
    })
    
    // Try to identify and keep only the main content
    c.extractMainContent(doc)
}

// extractMainContent attempts to identify and keep only the main content of the page
func (c *Converter) extractMainContent(doc *goquery.Document) {
    // Common selectors for main content
    mainContentSelectors := []string{
        "article", "main", ".post-content", ".article-content", ".entry-content", 
        "#content", ".content", ".post", ".article", ".entry", 
        "[itemprop='articleBody']", ".story", ".story-body",
    }
    
    // Try to find main content by common selectors
    var mainContent *goquery.Selection
    
    // First pass: try to find exact main content container
    for _, selector := range mainContentSelectors {
        selection := doc.Find(selector)
        if selection.Length() > 0 {
            // Check if this has substantial content
            if len(strings.TrimSpace(selection.Text())) > 200 {
                mainContent = selection
                break
            }
        }
    }
    
    // If we found a main content container, replace body with just that content
    if mainContent != nil && mainContent.Length() > 0 {
        // Create a new body with only the main content
        newBody := doc.Find("body").Empty()
        newBody.AppendSelection(mainContent.Clone())
        return
    }
    
    // If no main content container found, use a density-based approach
    // This keeps sections with high text-to-HTML ratio and removes others
    c.cleanByContentDensity(doc)
}

// cleanByContentDensity removes elements with low text-to-HTML ratio
func (c *Converter) cleanByContentDensity(doc *goquery.Document) {
    // Elements that should be evaluated for content density
    containers := doc.Find("div, section, article")
    
    containers.Each(func(i int, s *goquery.Selection) {
        // Skip if already removed or tiny
        if s.Length() == 0 || len(s.Text()) < 100 {
            return
        }
        
        html, _ := s.Html()
        text := strings.TrimSpace(s.Text())
        
        // Calculate text-to-HTML ratio (higher is better)
        textLength := len(text)
        htmlLength := len(html)
        
        if htmlLength > 0 {
            ratio := float64(textLength) / float64(htmlLength)
            
            // If ratio is too low (too much HTML, not enough text), 
            // it's likely not main content
            if ratio < 0.2 && len(s.Find("p, h1, h2, h3, h4, h5, h6").Text()) < 100 {
                // But make sure we're not removing an important element
                // that has too much formatting
                imgCount := s.Find("img").Length()
                linkCount := s.Find("a").Length()
                pCount := s.Find("p").Length()
                
                // Consider images in your decision - use imgCount
                if pCount < 3 && linkCount > pCount*2 && imgCount < 2 && len(text) < 500 {
                    s.Remove()
                }
            }
        }
    })
    
    // Final pass: remove empty containers
    doc.Find("div, section, article").Each(func(i int, s *goquery.Selection) {
        if len(strings.TrimSpace(s.Text())) == 0 && s.Find("img").Length() == 0 {
            s.Remove()
        }
    })
}

// Process an HTML element and convert it to Markdown
func (c *Converter) processElement(sb *strings.Builder, s *goquery.Selection, depth int) {
    // Either remove the tagName variable if not used
    // OR keep it if you plan to use it later
    
    // Check for inline styling first
    style, hasStyle := s.Attr("style")
    class, hasClass := s.Attr("class")
    
    // Track if we need to apply formatting
    isBold := false
    isItalic := false
    isHighlighted := false
    isStrikethrough := false
    isUnderlined := false
    
    // Check element type for formatting
    if s.Is("strong") || s.Is("b") {
        isBold = true
    }
    if s.Is("em") || s.Is("i") {
        isItalic = true
    }
    if s.Is("mark") {
        isHighlighted = true
    }
    if s.Is("s") || s.Is("strike") || s.Is("del") {
        isStrikethrough = true
    }
    if s.Is("u") || s.Is("ins") {
        isUnderlined = true
    }
    
    // Check CSS classes for common formatting patterns
    if hasClass {
        lcClass := strings.ToLower(class)
        if strings.Contains(lcClass, "bold") || strings.Contains(lcClass, "strong") || strings.Contains(lcClass, "heading") {
            isBold = true
        }
        if strings.Contains(lcClass, "italic") || strings.Contains(lcClass, "oblique") || strings.Contains(lcClass, "emphasis") {
            isItalic = true
        }
        if strings.Contains(lcClass, "highlight") || strings.Contains(lcClass, "mark") {
            isHighlighted = true
        }
    }
    
    // Check inline styles for formatting
    if hasStyle {
        lcStyle := strings.ToLower(style)
        if strings.Contains(lcStyle, "font-weight:") {
            weightValue := regexp.MustCompile(`font-weight:\s*(\d+|bold|bolder)`).FindStringSubmatch(lcStyle)
            if len(weightValue) > 1 {
                if weightValue[1] == "bold" || weightValue[1] == "bolder" || weightValue[1] >= "600" {
                    isBold = true
                }
            }
        }
        if strings.Contains(lcStyle, "font-style:") {
            styleValue := regexp.MustCompile(`font-style:\s*(italic|oblique)`).FindStringSubmatch(lcStyle)
            if len(styleValue) > 0 {
                isItalic = true
            }
        }
        if strings.Contains(lcStyle, "text-decoration:") {
            if strings.Contains(lcStyle, "underline") {
                isUnderlined = true
            }
            if strings.Contains(lcStyle, "line-through") {
                isStrikethrough = true
            }
        }
        if strings.Contains(lcStyle, "background-color:") && !strings.Contains(lcStyle, "background-color:transparent") {
            isHighlighted = true
        }
    }

    // Process block elements
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
        
        // Extract text while preserving formatting in headings
        var headingText strings.Builder
        s.Contents().Each(func(i int, child *goquery.Selection) {
            if child.Nodes != nil && len(child.Nodes) > 0 {
                if child.Nodes[0].Type == 1 { // Element node
                    var nestedContent strings.Builder
                    c.processElement(&nestedContent, child, depth+1)
                    headingText.WriteString(nestedContent.String())
                } else if child.Nodes[0].Type == 3 { // Text node
                    headingText.WriteString(child.Text())
                }
            }
        })
        
        sb.WriteString("\n" + strings.Repeat("#", level) + " " + strings.TrimSpace(headingText.String()) + "\n\n")
        
    } else if s.Is("p") {
        // Handle paragraphs, but preserve inline formatting
        var paraText strings.Builder
        s.Contents().Each(func(i int, child *goquery.Selection) {
            if child.Nodes != nil && len(child.Nodes) > 0 {
                if child.Nodes[0].Type == 1 { // Element node
                    c.processElement(&paraText, child, depth+1)
                } else if child.Nodes[0].Type == 3 { // Text node
                    paraText.WriteString(child.Text())
                }
            }
        })
        
        if paraText.Len() > 0 {
            sb.WriteString(strings.TrimSpace(paraText.String()) + "\n\n")
        }
        
    } else if s.Is("ul, ol") {
        // Handle lists with proper formatting
        sb.WriteString("\n")
        s.Find("li").Each(func(i int, li *goquery.Selection) {
            // Track list item formatting
            var listItemText strings.Builder
            
            // Process the list item content with formatting
            li.Contents().Each(func(j int, child *goquery.Selection) {
                if child.Nodes != nil && len(child.Nodes) > 0 {
                    if child.Nodes[0].Type == 1 { // Element node
                        c.processElement(&listItemText, child, depth+1)
                    } else if child.Nodes[0].Type == 3 { // Text node
                        listItemText.WriteString(child.Text())
                    }
                }
            })
            
            // Add the formatted list item
            if s.Is("ul") {
                sb.WriteString("- " + strings.TrimSpace(listItemText.String()) + "\n")
            } else {
                sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, strings.TrimSpace(listItemText.String())))
            }
        })
        sb.WriteString("\n")
        
    } else if s.Is("blockquote") {
        // Handle blockquotes with formatting
        var quoteText strings.Builder
        s.Contents().Each(func(i int, child *goquery.Selection) {
            if child.Nodes != nil && len(child.Nodes) > 0 {
                if child.Nodes[0].Type == 1 { // Element node
                    c.processElement(&quoteText, child, depth+1)
                } else if child.Nodes[0].Type == 3 { // Text node
                    quoteText.WriteString(child.Text())
                }
            }
        })
        
        // Split the quote by lines and add > to each line
        lines := strings.Split(strings.TrimSpace(quoteText.String()), "\n")
        for _, line := range lines {
            sb.WriteString("> " + line + "\n")
        }
        sb.WriteString("\n")
        
    } else if s.Is("a") {
        // Handle links
        href, exists := s.Attr("href")
        
        // Extract link text with formatting
        var linkText strings.Builder
        s.Contents().Each(func(i int, child *goquery.Selection) {
            if child.Nodes != nil && len(child.Nodes) > 0 {
                if child.Nodes[0].Type == 1 { // Element node
                    c.processElement(&linkText, child, depth+1)
                } else if child.Nodes[0].Type == 3 { // Text node
                    linkText.WriteString(child.Text())
                }
            }
        })
        
        text := strings.TrimSpace(linkText.String())
        if text == "" {
            text = strings.TrimSpace(s.Text())
        }
        
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
        // Handle line breaks
        sb.WriteString("\n")
        
    } else if s.Is("div") {
        // Handle div containers - process children with possible formatting
        if hasStyle && strings.Contains(style, "text-align:center") {
            // For centered text, add HTML comment since Markdown doesn't support alignment
            sb.WriteString("\n<div style=\"text-align:center\">\n")
            
            // Process children
            s.Contents().Each(func(i int, child *goquery.Selection) {
                if child.Nodes != nil && len(child.Nodes) > 0 {
                    if child.Nodes[0].Type == 1 { // Element node
                        c.processElement(sb, child, depth+1)  // FIXED: removed the & before sb
                    } else if child.Nodes[0].Type == 3 { // Text node
                        text := strings.TrimSpace(child.Text())
                        if text != "" {
                            sb.WriteString(text)
                        }
                    }
                }
            })
            
            sb.WriteString("\n</div>\n\n")
        } else {
            // Regular div - process children
            s.Contents().Each(func(i int, child *goquery.Selection) {
                if child.Nodes != nil && len(child.Nodes) > 0 {
                    if child.Nodes[0].Type == 1 { // Element node
                        c.processElement(sb, child, depth+1)  // FIXED: removed the & before sb
                    } else if child.Nodes[0].Type == 3 { // Text node
                        text := strings.TrimSpace(child.Text())
                        if text != "" {
                            sb.WriteString(text)
                        }
                    }
                }
            })
        }
        
    } else if s.Is("table") {
        // Handle tables (no change to your existing table handling)
        c.processTable(sb, s)
        
    } else {
        // Handle text nodes and apply formatting
        
        // Start formatting markers
        if isBold {
            sb.WriteString("**")
        }
        if isItalic {
            sb.WriteString("*")
        }
        if isHighlighted {
            sb.WriteString("==")
        }
        if isStrikethrough {
            sb.WriteString("~~")
        }
        if isUnderlined {
            sb.WriteString("<u>") // Obsidian supports HTML tags for underline
        }
        
        // Process child elements recursively
        hasContent := false
        s.Contents().Each(func(i int, child *goquery.Selection) {
            if child.Nodes != nil && len(child.Nodes) > 0 {
                if child.Nodes[0].Type == 1 { // Element node
                    hasContent = true
                    c.processElement(sb, child, depth+1)
                } else if child.Nodes[0].Type == 3 { // Text node
                    text := child.Text()
                    if text != "" {
                        hasContent = true
                        sb.WriteString(text)
                    }
                }
            }
        })
        
        // If no content, try to get text
        if !hasContent && s.Text() != "" {
            sb.WriteString(s.Text())
            hasContent = true
        }
        
        // Close formatting markers in reverse order
        if isUnderlined {
            sb.WriteString("</u>") 
        }
        if isStrikethrough {
            sb.WriteString("~~")
        }
        if isHighlighted {
            sb.WriteString("==")
        }
        if isItalic {
            sb.WriteString("*")
        }
        if isBold {
            sb.WriteString("**")
        }
    }
}

// Process HTML tables and convert to Markdown tables with preserved formatting
func (c *Converter) processTable(sb *strings.Builder, table *goquery.Selection) {
    sb.WriteString("\n")

    // Process table headers
    headers := []string{}
    table.Find("thead tr th").Each(func(i int, s *goquery.Selection) {
        // Parse header with formatting
        var headerContent strings.Builder
        c.processElement(&headerContent, s, 0)
        headers = append(headers, strings.TrimSpace(headerContent.String()))
    })

    // If no headers found in thead, check first tr
    if len(headers) == 0 {
        table.Find("tr:first-child th, tr:first-child td").Each(func(i int, s *goquery.Selection) {
            var headerContent strings.Builder
            c.processElement(&headerContent, s, 0)
            headers = append(headers, strings.TrimSpace(headerContent.String()))
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
            
            // Process cell content with formatting
            var cellContent strings.Builder
            c.processElement(&cellContent, cell, 0)
            sb.WriteString(strings.TrimSpace(cellContent.String()))
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