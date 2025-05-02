package scraper

import (
    "fmt"
    "math/rand"
    "net/http"
    "time"

    "github.com/PuerkitoBio/goquery"
)

// UserAgents contains a list of common browser user agents for rotation
var UserAgents = []string{
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:90.0) Gecko/20100101 Firefox/90.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:90.0) Gecko/20100101 Firefox/90.0",
}

// Scraper represents a web scraper with configuration for handling rate limits
type Scraper struct {
    URL              string
    MinDelay         time.Duration
    MaxDelay         time.Duration
    MaxRetries       int
    RetryBackoffBase float64
    client           *http.Client
}

// NewScraper creates a new scraper with default settings to prevent rate limiting
func NewScraper(url string) *Scraper {
    return &Scraper{
        URL:              url,
        MinDelay:         1 * time.Second,
        MaxDelay:         3 * time.Second,
        MaxRetries:       3,
        RetryBackoffBase: 2,
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// getRandomUserAgent returns a random user agent from the list
func getRandomUserAgent() string {
    return UserAgents[rand.Intn(len(UserAgents))]
}

// Fetch retrieves the HTML content from the URL with anti-detection measures
func (s *Scraper) Fetch() (*goquery.Document, error) {
    var (
        resp *http.Response
        err  error
    )

    // Initialize random number generator
    rand.Seed(time.Now().UnixNano())

    fmt.Printf("Fetching URL: %s\n", s.URL)

    // Implement exponential backoff for retries
    for attempt := 0; attempt <= s.MaxRetries; attempt++ {
        // Random delay between requests to avoid detection
        if attempt > 0 {
            delay := s.MinDelay
            if attempt > 1 {
                // Exponential backoff
                maxBackoff := time.Duration(float64(s.MinDelay) * s.RetryBackoffBase * float64(attempt-1))
                if maxBackoff > s.MaxDelay {
                    maxBackoff = s.MaxDelay
                }
                delay = time.Duration(rand.Int63n(int64(maxBackoff-s.MinDelay))) + s.MinDelay
            }
            fmt.Printf("Waiting %v before retry %d\n", delay, attempt)
            time.Sleep(delay)
        }

        // Create a new request
        req, err := http.NewRequest("GET", s.URL, nil)
        if err != nil {
            return nil, fmt.Errorf("error creating request: %w", err)
        }

        // Set random User-Agent to avoid detection
        userAgent := getRandomUserAgent()
        req.Header.Set("User-Agent", userAgent)
        fmt.Printf("Using User-Agent: %s\n", userAgent)
        
        // Add common headers to appear more like a normal browser
        req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
        req.Header.Set("Accept-Language", "en-US,en;q=0.5")
        req.Header.Set("Connection", "keep-alive")
        req.Header.Set("Upgrade-Insecure-Requests", "1")

        // Make the request
        resp, err = s.client.Do(req)
        if err != nil {
            fmt.Printf("Request error: %v (attempt %d/%d)\n", err, attempt+1, s.MaxRetries+1)
            continue
        }
        
        fmt.Printf("HTTP Status: %d\n", resp.StatusCode)
        
        if resp.StatusCode == http.StatusOK {
            break
        }

        if resp != nil {
            resp.Body.Close()
        }

        // If this was our last attempt, return the error
        if attempt == s.MaxRetries {
            if err != nil {
                return nil, fmt.Errorf("failed to fetch URL after %d retries: %w", s.MaxRetries, err)
            }
            return nil, fmt.Errorf("failed to fetch URL after %d retries: status code %d", s.MaxRetries, resp.StatusCode)
        }
    }

    // Parse the HTML document
    doc, err := goquery.NewDocumentFromReader(resp.Body)
    defer resp.Body.Close()
    if err != nil {
        return nil, fmt.Errorf("error parsing HTML: %w", err)
    }

    // Check if we actually got HTML content
    if doc.Find("html").Length() == 0 {
        fmt.Println("Warning: Response doesn't appear to be HTML")
    } else {
        fmt.Println("HTML document successfully parsed")
    }

    return doc, nil
}