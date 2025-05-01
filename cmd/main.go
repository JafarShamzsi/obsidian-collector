package main

import (
    "flag"
    "fmt"
    "log"
    "net/url"
    "os"
    "strings"
    "time"

    "obsidian-web-scraper/internal/config"
    "obsidian-web-scraper/internal/links"
    "obsidian-web-scraper/internal/markdown"
    "obsidian-web-scraper/internal/scraper"
    "obsidian-web-scraper/internal/storage"
)

func main() {
    // Define CLI flags
    configPath := flag.String("config", "", "Path to config file")
    url := flag.String("url", "", "URL to scrape (overrides config file)")
    outputDir := flag.String("output", "", "Output directory (overrides config file)")
    folder := flag.String("folder", "Web Clippings", "Obsidian folder to save in")
    selector := flag.String("selector", "", "CSS selector for content (overrides config file)")
    depth := flag.Int("depth", 1, "Maximum recursion depth for following links (0=no recursion)")
    maxPages := flag.Int("max-pages", 10, "Maximum number of pages to scrape in recursive mode")
    
    flag.Parse()
    
    // Load configuration
    cfg, err := config.LoadConfig(*configPath)
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }
    
    // Override config with command line arguments if provided
    if *url != "" {
        cfg.TargetURL = *url
    }
    if *outputDir != "" {
        cfg.OutputDir = *outputDir
    }
    if *selector != "" {
        cfg.ContentSelector = *selector
    }
    
    // Validate URL
    if cfg.TargetURL == "" {
        log.Fatal("Target URL is required. Provide it via config file or --url flag")
    }
    
    // Create base URL for link resolution
    baseURL := cfg.TargetURL
    parsedURL, err := url.Parse(baseURL)
    if err == nil {
        baseURL = parsedURL.Scheme + "://" + parsedURL.Host
    }

    // Initialize link manager for tracking connections between pages
    linkManager := links.NewLinkManager(baseURL)
    
    // Initialize file manager
    fileManager := storage.NewFileManager(cfg.OutputDir)
    
    // Create a queue of URLs to scrape
    urlsToScrape := []string{cfg.TargetURL}
    scrapedCount := 0
    
    // Start scraping
    for len(urlsToScrape) > 0 && scrapedCount < *maxPages {
        // Get next URL from queue
        currentURL := urlsToScrape[0]
        urlsToScrape = urlsToScrape[1:]
        
        fmt.Printf("Scraping URL (%d/%d): %s\n", scrapedCount+1, *maxPages, currentURL)
        
        // Scrape the page
        filePath, err := scrapePage(currentURL, cfg, fileManager, linkManager, *folder)
        if err != nil {
            log.Printf("Failed to scrape %s: %v\n", currentURL, err)
            continue
        }
        
        scrapedCount++
        fmt.Printf("Successfully saved to: %s\n", filePath)
        
        // If we've reached maximum depth, don't add more URLs to queue
        if *depth > 0 && scrapedCount < *maxPages {
            // Get pending URLs from the link manager
            pendingURLs := linkManager.GetPendingURLs()
            linkManager.ClearPendingURLs()
            
            // Add unique new URLs to the queue
            for _, pendingURL := range pendingURLs {
                urlsToScrape = append(urlsToScrape, pendingURL)
            }
            
            fmt.Printf("Found %d links to follow\n", len(pendingURLs))
        }
        
        // Add random delay between page scrapes to avoid rate limiting
        if len(urlsToScrape) > 0 {
            delay := time.Duration(cfg.MinDelay) * time.Millisecond
            fmt.Printf("Waiting %.1f seconds before next page...\n", 
                float64(delay)/float64(time.Second))
            time.Sleep(delay)
        }
    }
    
    fmt.Printf("Scraping complete! Processed %d pages.\n", scrapedCount)
}

// scrapePage handles scraping a single page and saving it to a file
func scrapePage(targetURL string, cfg *config.Config, fileManager *storage.FileManager, linkManager *links.LinkManager, folder string) (string, error) {
    // Initialize scraper with rate-limiting settings
    s := scraper.NewScraper(targetURL)
    s.MinDelay = time.Duration(cfg.MinDelay) * time.Millisecond
    s.MaxDelay = time.Duration(cfg.MaxDelay) * time.Millisecond
    s.MaxRetries = cfg.MaxRetries
    
    // Fetch the content
    doc, err := s.Fetch()
    if err != nil {
        return "", fmt.Errorf("failed to fetch content: %v", err)
    }
    
    // Parse the content
    content, err := scraper.Parse(doc, cfg.ContentSelector)
    if err != nil {
        return "", fmt.Errorf("failed to parse content: %v", err)
    }
    
    // Create base URL from target URL (for resolving relative links)
    baseURL := targetURL
    if i := strings.Index(baseURL, "://"); i > 0 {
        hostEnd := strings.Index(baseURL[i+3:], "/")
        if hostEnd > 0 {
            baseURL = baseURL[:i+3+hostEnd]
        }
    }
    
    // Convert to markdown
    converter := markdown.NewConverter(baseURL, linkManager)
    md, err := converter.HTMLToMarkdown(content.Title, content.Content, content.Metadata)
    if err != nil {
        return "", fmt.Errorf("failed to convert content to markdown: %v", err)
    }
    
    // Save to file
    filePath, err := fileManager.SaveMarkdown(md, content.Title, folder)
    if err != nil {
        return "", fmt.Errorf("failed to save markdown file: %v", err)
    }
    
    // Register this page in the link manager
    linkManager.RegisterFile(targetURL, filePath)
    
    return filePath, nil
}