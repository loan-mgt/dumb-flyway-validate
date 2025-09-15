package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/loan-mgt/dumb-flyway-validate/internal/git"
	"github.com/loan-mgt/dumb-flyway-validate/pkg/errors"
)

// Validator interface for migration validation
type Validator interface {
	Validate(changes []git.FileChange) ([]errors.ValidationError, error)
}

// MigrationValidator implements validation for Flyway migrations
type MigrationValidator struct {
	migrationDir string
	debug        bool
}

// NewMigrationValidator creates a new migration validator
func NewMigrationValidator(migrationDir string, debug bool) *MigrationValidator {
	return &MigrationValidator{
		migrationDir: migrationDir,
		debug:        debug,
	}
}

// Validate validates file changes against Flyway migration rules
func (v *MigrationValidator) Validate(changes []git.FileChange) ([]errors.ValidationError, error) {
	var validationErrors []errors.ValidationError

	if v.debug {
		fmt.Printf("Starting validation of %d file changes\n", len(changes))
	}

	for i, change := range changes {
		if v.debug {
			fmt.Printf("\n--- Processing file %d/%d ---\n", i+1, len(changes))
			fmt.Printf("File: %s\n", change.NewPath)
			fmt.Printf("Status: %s\n", change.Status)
			if change.OldPath != change.NewPath {
				fmt.Printf("Old path: %s\n", change.OldPath)
			}
		}

		relativeFilePath, err := filepath.Rel(v.migrationDir, change.NewPath)
		if err != nil {
			if v.debug {
				fmt.Printf("Error getting relative path: %v\n", err)
			}
			return nil, fmt.Errorf("failed to get relative path for %s: %w", change.NewPath, err)
		}

		if v.debug {
			fmt.Printf("Relative path: %s\n", relativeFilePath)
		}

		isMigration := v.isMigrationFile(relativeFilePath)
		if v.debug {
			fmt.Printf("Is migration file: %t\n", isMigration)
		}

		// Skip non-migration files
		if !isMigration {
			if v.debug {
				fmt.Printf("Skipping non-migration file\n")
			}
			continue
		}

		isLast := v.isAlphabeticallyLast(relativeFilePath)
		if v.debug {
			fmt.Printf("Is alphabetically last: %t\n", isLast)
		}

		validationError := v.validateChange(change, relativeFilePath)
		if validationError != nil {
			if v.debug {
				fmt.Printf("VALIDATION ERROR: %s\n", validationError.Message)
			}
			validationErrors = append(validationErrors, *validationError)
		} else if v.debug {
			fmt.Printf("No validation errors for this file\n")
		}
	}

	if v.debug {
		fmt.Printf("\nValidation completed. Total errors: %d\n", len(validationErrors))
	}

	return validationErrors, nil
}

// validateChange validates a single file change
func (v *MigrationValidator) validateChange(change git.FileChange, relativeFilePath string) *errors.ValidationError {
	if v.debug {
		fmt.Printf("Validating change for file: %s\n", relativeFilePath)
		fmt.Printf("Change status: %s\n", change.Status)
	}

	switch {
	case v.isModified(change.Status):
		if v.debug {
			fmt.Printf("File was modified - this is not allowed for migration files\n")
		}
		return &errors.ValidationError{
			Type:    errors.ErrorTypeModified,
			Message: "Cannot modify migration file after it was applied",
			OldPath: change.OldPath,
			NewPath: change.NewPath,
		}
	case v.isAdded(change.Status):
		if v.debug {
			fmt.Printf("File was added - checking if it's alphabetically last\n")
		}
		if !v.isAlphabeticallyLast(relativeFilePath) {
			if v.debug {
				fmt.Printf("Added file is NOT alphabetically last - validation error\n")
			}
			return &errors.ValidationError{
				Type:    errors.ErrorTypeNotLast,
				Message: "Added migration file not alphabetically last",
				OldPath: change.OldPath,
				NewPath: change.NewPath,
			}
		} else if v.debug {
			fmt.Printf("Added file is alphabetically last - OK\n")
		}
	case v.isDeleted(change.Status):
		if v.debug {
			fmt.Printf("File was deleted - this is not allowed for migration files\n")
		}
		return &errors.ValidationError{
			Type:    errors.ErrorTypeRemoved,
			Message: "Cannot remove migration file after it was applied",
			OldPath: change.OldPath,
			NewPath: change.NewPath,
		}
	case v.isRenamed(change.Status):
		if v.debug {
			fmt.Printf("File was renamed - this is not allowed for migration files\n")
		}
		return &errors.ValidationError{
			Type:    errors.ErrorTypeRenamed,
			Message: "Cannot rename migration file after it was applied",
			OldPath: change.OldPath,
			NewPath: change.NewPath,
		}
	default:
		if v.debug {
			fmt.Printf("Unknown or unhandled status: %s\n", change.Status)
		}
	}

	return nil
}

// isMigrationFile checks if a file is a Flyway migration file
func (v *MigrationValidator) isMigrationFile(filePath string) bool {
	// Improved regex to match full Flyway pattern: V{version}__{description}.sql
	re := regexp.MustCompile(`^V\d+(\.\d+)*__\w+.*\.sql$`)
	fileName := filepath.Base(filePath)
	result := re.MatchString(fileName)
	
	if v.debug {
		fmt.Printf("Checking if '%s' is a migration file: %t\n", fileName, result)
		fmt.Printf("  Migration file pattern: V{version}__{description}.sql\n")
	}
	
	return result
}

// isAlphabeticallyLast checks if the migration file is alphabetically last
func (v *MigrationValidator) isAlphabeticallyLast(filePath string) bool {
	files, err := v.getMigrationFiles()
	if err != nil {
		if v.debug {
			fmt.Printf("Error getting migration files: %v\n", err)
		}
		return false
	}

	if v.debug {
		fmt.Printf("Found %d existing migration files:\n", len(files))
		for i, file := range files {
			fmt.Printf("  [%d] %s\n", i+1, filepath.Base(file))
		}
	}

	sort.Strings(files)

	// Check if this file would be last when added to the sorted list
	fileName := filepath.Base(filePath)
	if v.debug {
		fmt.Printf("Checking if '%s' would be alphabetically last\n", fileName)
	}

	for i, file := range files {
		fileBase := filepath.Base(file)
		if fileName <= fileBase {
			result := i == len(files)-1 && fileName == fileBase
			if v.debug {
				fmt.Printf("  Compared with existing file '%s': fileName <= fileBase: %t\n", fileBase, fileName <= fileBase)
				fmt.Printf("  Is last and equal: %t\n", result)
			}
			return result
		}
	}

	// If we get here, the file would be added at the end
	if v.debug {
		fmt.Printf("  File would be added at the end (alphabetically last): true\n")
	}
	return true
}

// getMigrationFiles returns all migration files in the migration directory
func (v *MigrationValidator) getMigrationFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(v.migrationDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && v.isMigrationFile(path) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// Git status helpers
func (v *MigrationValidator) isModified(status string) bool {
	return v.matchStatus(status, `^M`)
}

func (v *MigrationValidator) isAdded(status string) bool {
	return v.matchStatus(status, `^A`)
}

func (v *MigrationValidator) isDeleted(status string) bool {
	return v.matchStatus(status, `^D`)
}

func (v *MigrationValidator) isRenamed(status string) bool {
	return v.matchStatus(status, `^R`)
}

func (v *MigrationValidator) matchStatus(status, pattern string) bool {
	re := regexp.MustCompile(pattern)
	return re.MatchString(strings.ToUpper(status))
}
