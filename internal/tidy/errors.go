package tidy

import (
	"fmt"
)

// ErrorType represents different types of errors that can occur during tidy operations.
type ErrorType int

const (
	// ErrorTypeFilesystem indicates a file system operation failed.
	ErrorTypeFilesystem ErrorType = iota
	// ErrorTypeRegistry indicates a registry operation failed.
	ErrorTypeRegistry
	// ErrorTypeInvalidPath indicates an invalid path was provided.
	ErrorTypeInvalidPath
)

// TidyError represents an error that occurs during tidy operations.
type TidyError struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *TidyError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *TidyError) Unwrap() error {
	return e.Err
}

func (e *TidyError) Is(target error) bool {
	t, ok := target.(*TidyError)
	if !ok {
		return false
	}
	return e.Type == t.Type
}
