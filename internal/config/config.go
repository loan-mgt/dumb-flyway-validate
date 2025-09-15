package config

import (
	"flag"
	"fmt"
	"os"
)

// Config holds all configuration for the application
type Config struct {
	MigrationDir  string
	CompareBranch string
	Debug         bool
	Integration   bool
	Help          bool
	GitLab        GitLabConfig
}

// GitLabConfig holds GitLab-specific configuration
type GitLabConfig struct {
	URL       string
	Token     string
	ProjectID string
	MRIID     string
}

// LoadConfig parses command line flags and environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// Parse command line flags
	flag.StringVar(&cfg.MigrationDir, "migration-dir", ".", "Specify the migration directory (default: current directory)")
	flag.StringVar(&cfg.CompareBranch, "branch", "", "Specify the branch to compare against (default: empty, i.e., working directory)")
	flag.BoolVar(&cfg.Debug, "debug", false, "Enable debug mode")
	flag.BoolVar(&cfg.Integration, "integration", false, "Enable integration with GitLab merge request interface")
	flag.BoolVar(&cfg.Help, "help", false, "Show help information")
	flag.Parse()

	// Handle help flag
	if cfg.Help {
		showHelp()
		os.Exit(0)
	}

	// Load GitLab configuration from environment variables
	cfg.GitLab = GitLabConfig{
		URL:       os.Getenv("GITLAB_URL"),
		Token:     os.Getenv("GITLAB_TOKEN"),
		ProjectID: os.Getenv("CI_PROJECT_ID"),
		MRIID:     os.Getenv("CI_MERGE_REQUEST_IID"),
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Check if migration directory exists
	if _, err := os.Stat(c.MigrationDir); os.IsNotExist(err) {
		return fmt.Errorf("migration directory does not exist: %s", c.MigrationDir)
	}

	// Validate GitLab configuration if integration is enabled
	if c.Integration {
		if c.GitLab.URL == "" || c.GitLab.Token == "" || c.GitLab.ProjectID == "" || c.GitLab.MRIID == "" {
			return fmt.Errorf("GitLab integration enabled but required environment variables are missing")
		}
	}

	return nil
}

// Print prints the configuration (for debug mode)
func (c *Config) Print() {
	if c.Debug {
		fmt.Println("Debug mode enabled")
		fmt.Println("Migration directory:", c.MigrationDir)
		fmt.Println("Compare branch:", c.CompareBranch)
		fmt.Println("Integration enabled:", c.Integration)

		if c.Integration {
			fmt.Println("GitLab URL:", c.GitLab.URL)
			fmt.Println("GitLab Token: [REDACTED]")
			fmt.Println("CI Project ID:", c.GitLab.ProjectID)
			fmt.Println("CI Merge Request IID:", c.GitLab.MRIID)
		}
	}
}

// showHelp displays the help information
func showHelp() {
	fmt.Println("dumb-flyway-validate - A tool to validate Flyway migration files")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  dumb-flyway-validate [OPTIONS]")
	fmt.Println()
	fmt.Println("OPTIONS:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("ENVIRONMENT VARIABLES (for GitLab integration):")
	fmt.Println("  GITLAB_URL              GitLab instance URL")
	fmt.Println("  GITLAB_TOKEN            GitLab API token")
	fmt.Println("  CI_PROJECT_ID           GitLab project ID")
	fmt.Println("  CI_MERGE_REQUEST_IID    GitLab merge request IID")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  dumb-flyway-validate --migration-dir=./migrations")
	fmt.Println("  dumb-flyway-validate --debug --branch=main")
	fmt.Println("  dumb-flyway-validate --integration --migration-dir=./db/migrations")
}
