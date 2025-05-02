package config

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
    TargetURL       string
    OutputDir       string
    MinDelay        int // Delay in milliseconds
    MaxDelay        int // Delay in milliseconds
    MaxRetries      int
    ContentSelector string // Optional CSS selector to target specific content
    RecursionDepth  int    // How many levels of links to follow
    MaxPages        int    // Maximum number of pages to scrape
    
    // Content cleaning options
    RemoveAds          bool // Remove advertising elements
    RemoveNavigation   bool // Remove navigation elements
    RemoveFooters      bool // Remove footer elements
    ExtractMainContent bool // Try to extract only the main content
}

// LoadConfig loads configuration from YAML file
func LoadConfig(configPath string) (*Config, error) {
    // Set defaults
    viper.SetDefault("min_delay", 1000)
    viper.SetDefault("max_delay", 3000)
    viper.SetDefault("max_retries", 3)
    viper.SetDefault("output_dir", "./output")
    viper.SetDefault("recursion_depth", 1)
    viper.SetDefault("max_pages", 10)
    viper.SetDefault("remove_ads", true)
    viper.SetDefault("remove_navigation", true)
    viper.SetDefault("remove_footers", true) 
    viper.SetDefault("extract_main_content", true)
    
    // If config file is explicitly provided
    if configPath != "" {
        viper.SetConfigFile(configPath)
    } else {
        // Look for config in the standard location
        viper.SetConfigName("config")
        viper.SetConfigType("yaml")
        viper.AddConfigPath("./config")
    }

    // Read the config file
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            // Config file not found; ignore error if using default values
            fmt.Println("Warning: No config file found, using default settings")
        } else {
            // Config file was found but another error occurred
            return nil, fmt.Errorf("error reading config file: %w", err)
        }
    }

    cfg := &Config{
        TargetURL:          viper.GetString("target_url"),
        OutputDir:          viper.GetString("output_dir"),
        MinDelay:           viper.GetInt("min_delay"),
        MaxDelay:           viper.GetInt("max_delay"),
        MaxRetries:         viper.GetInt("max_retries"),
        ContentSelector:    viper.GetString("content_selector"),
        RecursionDepth:     viper.GetInt("recursion_depth"),
        MaxPages:           viper.GetInt("max_pages"),
        RemoveAds:          viper.GetBool("remove_ads"),
        RemoveNavigation:   viper.GetBool("remove_navigation"),
        RemoveFooters:      viper.GetBool("remove_footers"),
        ExtractMainContent: viper.GetBool("extract_main_content"),
    }

    // Validate required configuration
    if cfg.TargetURL == "" {
        return nil, fmt.Errorf("target_url is required in the configuration")
    }

    // Process output directory path 
    cfg.OutputDir = expandHomePath(cfg.OutputDir)

    // Ensure output directory exists
    if cfg.OutputDir != "" {
        absPath, err := filepath.Abs(cfg.OutputDir)
        if err != nil {
            return nil, fmt.Errorf("error resolving output directory path: %w", err)
        }
        
        if _, err := os.Stat(absPath); os.IsNotExist(err) {
            if err := os.MkdirAll(absPath, 0755); err != nil {
                return nil, fmt.Errorf("error creating output directory: %w", err)
            }
        }
        
        cfg.OutputDir = absPath
    }

    return cfg, nil
}

// Add this function to the config package
// expandHomePath expands the tilde character to user's home directory
func expandHomePath(path string) string {
    if strings.HasPrefix(path, "~/") {
        home, err := os.UserHomeDir()
        if err == nil {
            return filepath.Join(home, path[2:])
        }
    }
    return path
}