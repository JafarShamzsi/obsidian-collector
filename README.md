# Obsidian Web Scraper

This project is a web scraper designed to fetch content from a specified URL and convert it into markdown format suitable for Obsidian. The generated markdown files are saved to a user-defined directory.

## Project Structure

```
obsidian-web-scraper
├── cmd
│   └── main.go            # Entry point of the application
├── internal
│   ├── scraper
│   │   ├── scraper.go     # Scraper logic for fetching HTML content
│   │   └── parser.go      # Functions for parsing HTML content
│   ├── markdown
│   │   └── converter.go    # Functions to convert parsed data to markdown
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
   git clone https://github.com/yourusername/obsidian-web-scraper.git
   cd obsidian-web-scraper
   ```

2. Install the necessary dependencies:
   ```
   go mod tidy
   ```

3. Configure the application by editing the `config/config.yaml` file to specify the target URL and output directory.

## Usage

To run the application, execute the following command:
```
go run cmd/main.go
```

This will start the scraper, fetch the content from the specified URL, parse it, convert it to markdown, and save it to the designated folder.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## License

This project is licensed under the MIT License. See the LICENSE file for more details.