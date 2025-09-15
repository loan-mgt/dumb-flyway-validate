package errors

import "fmt"

// ValidationErrorType represents the type of validation error
type ValidationErrorType int

const (
	ErrorTypeModified ValidationErrorType = iota
	ErrorTypeNotLast
	ErrorTypeRemoved
	ErrorTypeRenamed
)

// String returns a string representation of the error type
func (e ValidationErrorType) String() string {
	switch e {
	case ErrorTypeModified:
		return "MODIFIED"
	case ErrorTypeNotLast:
		return "NOT_LAST"
	case ErrorTypeRemoved:
		return "REMOVED"
	case ErrorTypeRenamed:
		return "RENAMED"
	default:
		return "UNKNOWN"
	}
}

// ValidationError represents a migration validation error
type ValidationError struct {
	Type    ValidationErrorType
	Message string
	OldPath string
	NewPath string
	Details map[string]string
}

// Error implements the error interface
func (e ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Message, e.NewPath)
}

// MarkdownMessage returns the error message formatted for GitLab markdown
func (e ValidationError) MarkdownMessage() string {
	switch e.Type {
	case ErrorTypeModified:
		return ":warning: Error: Cannot **modify** migration file after it was applied"
	case ErrorTypeNotLast:
		return ":warning: Error: Added migration file **not** alphabetically **last**"
	case ErrorTypeRemoved:
		return ":warning: Error: Cannot **remove** migration file after it was applied"
	case ErrorTypeRenamed:
		return ":warning: Error: Cannot **rename** migration file after it was applied"
	default:
		return ":warning: Error: " + e.Message
	}
}

// NewValidationError creates a new validation error
func NewValidationError(errType ValidationErrorType, message, oldPath, newPath string) ValidationError {
	return ValidationError{
		Type:    errType,
		Message: message,
		OldPath: oldPath,
		NewPath: newPath,
		Details: make(map[string]string),
	}
}

// NewModifiedError creates a validation error for modified migrations
func NewModifiedError(oldPath, newPath string) ValidationError {
	return NewValidationError(
		ErrorTypeModified,
		"Cannot modify migration file after it was applied",
		oldPath,
		newPath,
	)
}

// NewNotLastError creates a validation error for migrations not alphabetically last
func NewNotLastError(oldPath, newPath string) ValidationError {
	return NewValidationError(
		ErrorTypeNotLast,
		"Added migration file not alphabetically last",
		oldPath,
		newPath,
	)
}

// NewRemovedError creates a validation error for removed migrations
func NewRemovedError(oldPath, newPath string) ValidationError {
	return NewValidationError(
		ErrorTypeRemoved,
		"Cannot remove migration file after it was applied",
		oldPath,
		newPath,
	)
}

// NewRenamedError creates a validation error for renamed migrations
func NewRenamedError(oldPath, newPath string) ValidationError {
	return NewValidationError(
		ErrorTypeRenamed,
		"Cannot rename migration file after it was applied",
		oldPath,
		newPath,
	)
}
