package links

import (
    "net/url"
    "path/filepath"
    "strings"
    "sync"
)

// LinkManager tracks relationships between web URLs and their corresponding Obsidian files
type LinkManager struct {
    baseURL     string
    urlToFile   map[string]string
    mutex       sync.RWMutex
    scraped     map[string]bool
    pendingURLs []string
}

// NewLinkManager creates a new link manager for a specific website
func NewLinkManager(baseURL string) *LinkManager {
    // Ensure the base URL ends with a slash for proper path joining
    if !strings.HasSuffix(baseURL, "/") {
        baseURL = baseURL + "/"
    }
    
    return &LinkManager{
        baseURL:   baseURL,
        urlToFile: make(map[string]string),
        scraped:   make(map[string]bool),
        pendingURLs: []string{},
    }
}

// RegisterFile associates a URL with its corresponding markdown filename
func (lm *LinkManager) RegisterFile(url, filename string) {
    lm.mutex.Lock()
    defer lm.mutex.Unlock()
    
    lm.urlToFile[url] = filename
    lm.scraped[url] = true
}

// GetObsidianLink converts a web URL to an Obsidian internal link format if possible
func (lm *LinkManager) GetObsidianLink(href, linkText string) (string, bool) {
    lm.mutex.RLock()
    defer lm.mutex.RUnlock()
    
    // If it's not an internal link for the same site, keep it as an external link
    if !lm.isInternalLink(href) {
        return "", false
    }
    
    // Normalize the URL to handle relative paths
    fullURL := lm.normalizeURL(href)
    
    // If we've already scraped this page, use its registered filename
    if filename, exists := lm.urlToFile[fullURL]; exists {
        // Get basename without extension for Obsidian link
        basename := filepath.Base(filename)
        basename = strings.TrimSuffix(basename, filepath.Ext(basename))
        
        // Use link text if available, otherwise use the filename
        if linkText == "" {
            linkText = basename
        }
        
        return "[[" + basename + "|" + linkText + "]]", true
    }
    
    // If not yet scraped, add to pending URLs list for potential future scraping
    lm.addPendingURL(fullURL)
    
    // For now, return false to indicate we don't have an Obsidian link yet
    return "", false
}

// AddPendingURL adds a URL to the list of pending URLs that should be scraped
func (lm *LinkManager) addPendingURL(url string) {
    lm.mutex.Lock()
    defer lm.mutex.Unlock()
    
    // Only add if not already scraped or pending
    if !lm.scraped[url] {
        lm.pendingURLs = append(lm.pendingURLs, url)
        lm.scraped[url] = true // Mark as "known" to avoid duplicates
    }
}

// GetPendingURLs returns the list of URLs waiting to be scraped
func (lm *LinkManager) GetPendingURLs() []string {
    lm.mutex.RLock()
    defer lm.mutex.RUnlock()
    
    result := make([]string, len(lm.pendingURLs))
    copy(result, lm.pendingURLs)
    return result
}

// ClearPendingURLs clears the list of pending URLs
func (lm *LinkManager) ClearPendingURLs() {
    lm.mutex.Lock()
    defer lm.mutex.Unlock()
    
    lm.pendingURLs = []string{}
}

// IsInternalLink checks if a URL belongs to the same website
func (lm *LinkManager) isInternalLink(href string) bool {
    // Absolute URLs that start with the base URL
    if strings.HasPrefix(href, lm.baseURL) {
        return true
    }
    
    // Relative URLs (starting with / or without protocol)
    if strings.HasPrefix(href, "/") || !strings.Contains(href, "://") {
        return true
    }
    
    // Otherwise it's an external link
    return false
}

// NormalizeURL converts relative URLs to absolute
func (lm *LinkManager) normalizeURL(href string) string {
    // If already absolute, return as is
    if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
        return href
    }
    
    // Handle relative URLs
    if strings.HasPrefix(href, "/") {
        // Get base domain
        parsedURL, err := url.Parse(lm.baseURL)
        if err != nil {
            return href // Return original if parsing fails
        }
        
        baseWithScheme := parsedURL.Scheme + "://" + parsedURL.Host
        return baseWithScheme + href
    }
    
    // Other relative URLs
    return lm.baseURL + href
}