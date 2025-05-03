# Obsidian Collector

This project is a web scraper designed to fetch content from specified URLs and convert it into markdown format suitable for Obsidian. The generated markdown files are saved to a user-defined directory, preserving formatting and creating internal links where possible.

## Features

- Fetches web content and converts it to Obsidian-compatible markdown
- Preserves text formatting (bold, italic, headings, tables, etc.)
- Creates internal Obsidian links between scraped pages
- Cleans content by removing ads, navigation elements, and other clutter
- Supports recursive scraping to follow links on a page
- Configurable rate limiting to avoid overloading target websites
- Supports custom CSS selectors to target specific content areas

## Project Structure

```
obsidian-collector
├── cmd
│   └── main.go            # Entry point of the application
├── internal
│   ├── scraper
│   │   ├── scraper.go     # Scraper logic for fetching HTML content
│   │   └── parser.go      # Functions for parsing HTML content
│   ├── markdown
│   │   └── converter.go   # Functions to convert parsed data to markdown
│   ├── links
│   │   └── manager.go     # Manages link relationships between pages
│   ├── config
│   │   └── config.go      # Configuration loading and validation
│   └── storage
│       └── file_manager.go # Manages file operations for saving markdown
├── config
│   └── config.yaml        # Configuration settings for the application
├── go.mod                 # Module dependencies and Go version
├── go.sum                 # Checksums for module dependencies
└── README.md              # Documentation for the project
```

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/yourusername/obsidian-collector.git
   cd obsidian-collector
   ```

2. Install the necessary dependencies:
   ```
   go mod tidy
   ```

3. Configure the application by editing the config.yaml file to specify default settings.

## Usage

To run the application, execute the following command with your desired options:

```
go run cmd/main.go --url "https://example.com/article" --output "/path/to/obsidian/vault" --folder "Web Clippings"
```

### Command Line Options

- `--url`: URL to scrape (overrides config file)
- `--output`: Output directory for markdown files (overrides config file)
- `--folder`: Obsidian folder to save in (default: "Web Clippings")
- `--selector`: CSS selector for targeting specific content (overrides config file)
- `--depth`: Maximum recursion depth for following links (0=no recursion, default: 1)
- `--max-pages`: Maximum number of pages to scrape in recursive mode (default: 10)
- `--config`: Path to a custom config file

### Configuration File

You can customize default settings in the config.yaml file:

```yaml
# Web Scraper Configuration

# The URL to scrape
target_url: "https://example.com"

# Output directory for markdown files
output_dir: "~/Documents/Obsidian/MyVault"

# Rate limiting settings
min_delay: 1000  # Minimum delay between requests in milliseconds
max_delay: 3000  # Maximum delay between requests in milliseconds
max_retries: 3   # Maximum number of retry attempts 

# Optional: CSS selector for targeting specific content
content_selector: ""

# Recursive scraping settings
recursion_depth: 1   # How many levels of links to follow (0=none)
max_pages: 10        # Maximum number of pages to scrape

# Content cleaning settings
remove_ads: true     # Remove advertising elements
remove_navigation: true  # Remove navigation elements
remove_footers: true     # Remove footer elements
extract_main_content: true  # Try to extract only the main content
```

## Examples

### Scrape a single page:

```
go run cmd/main.go --url "https://example.com/article" --output "/path/to/vault" --folder "Articles"
```

### Scrape with recursive link following:

```
go run cmd/main.go --url "https://example.com/article" --output "/path/to/vault" --depth 2 --max-pages 20
```

### Target specific content with CSS selector:

```
go run cmd/main.go --url "https://example.com/article" --output "/path/to/vault" --selector ".article-content"
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## License

This project is licensed under the MIT License. See the LICENSE file for more details.