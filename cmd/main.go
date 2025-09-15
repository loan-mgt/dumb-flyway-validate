package main

import (
	"fmt"
	"os"

	"github.com/loan-mgt/dumb-flyway-validate/internal/config"
	"github.com/loan-mgt/dumb-flyway-validate/internal/git"
	"github.com/loan-mgt/dumb-flyway-validate/internal/gitlab"
	"github.com/loan-mgt/dumb-flyway-validate/internal/validator"
	"github.com/loan-mgt/dumb-flyway-validate/pkg/errors"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		os.Exit(1)
	}

	cfg.Print()

	if cfg.Debug {
		fmt.Println("\n=== INITIALIZING DEPENDENCIES ===")
	}

	// Initialize dependencies
	repo := git.NewLocalRepository()
	repo.SetDebug(cfg.Debug)
	migrationValidator := validator.NewMigrationValidator(cfg.MigrationDir, cfg.Debug)

	if cfg.Debug {
		fmt.Printf("Initialized local git repository\n")
		fmt.Printf("Initialized migration validator for directory: %s\n", cfg.MigrationDir)
		fmt.Println("\n=== GETTING GIT CHANGES ===")
	}

	// Get file changes from git
	changes, err := repo.GetChanges(cfg.MigrationDir, cfg.CompareBranch)
	if err != nil {
		fmt.Printf("Error getting git changes: %v\n", err)
		os.Exit(1)
	}

	if cfg.Debug {
		fmt.Printf("Found %d file changes\n", len(changes))
		for i, change := range changes {
			fmt.Printf("  [%d] Status: %s, Old: %s, New: %s\n", i+1, change.Status, change.OldPath, change.NewPath)
		}
		fmt.Println("\n=== VALIDATING CHANGES ===")
	}

	// Validate changes
	validationErrors, err := migrationValidator.Validate(changes)
	if err != nil {
		fmt.Printf("Error during validation: %v\n", err)
		os.Exit(1)
	}

	if cfg.Debug {
		fmt.Printf("Validation completed. Found %d validation errors\n", len(validationErrors))
		if len(validationErrors) > 0 {
			fmt.Println("Validation errors details:")
			for i, validationError := range validationErrors {
				fmt.Printf("  [%d] Type: %s, Message: %s\n", i+1, validationError.Type, validationError.Message)
				fmt.Printf("      Old Path: %s\n", validationError.OldPath)
				fmt.Printf("      New Path: %s\n", validationError.NewPath)
			}
		}
		fmt.Println("\n=== PROCESSING VALIDATION RESULTS ===")
	}

	// Handle validation results
	if len(validationErrors) > 0 {
		fmt.Println("Validation errors:")
		for _, validationError := range validationErrors {
			fmt.Printf("%s\n	%s\n\n", validationError.Error(), validationError.NewPath)
		}

		// Integrate with GitLab if enabled
		if cfg.Integration {
			if cfg.Debug {
				fmt.Println("=== GITLAB INTEGRATION ===")
				fmt.Println("GitLab integration is enabled, posting validation errors...")
			}
			if err := integrateWithGitLab(cfg, validationErrors); err != nil {
				fmt.Printf("Warning: GitLab integration failed: %v\n", err)
			} else if cfg.Debug {
				fmt.Println("Successfully posted validation errors to GitLab")
			}
		} else if cfg.Debug {
			fmt.Println("GitLab integration is disabled")
		}

		os.Exit(1)
	}

	if cfg.Debug {
		fmt.Println("No validation errors found!")
	}
	fmt.Println("Validation successful")
}

// integrateWithGitLab posts validation errors to GitLab merge request
func integrateWithGitLab(cfg *config.Config, validationErrors []errors.ValidationError) error {
	client := gitlab.NewClient(&cfg.GitLab)
	return client.PostValidationErrors(validationErrors)
}
