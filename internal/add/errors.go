package add

import (
	"fmt"
)

type ErrorType int

const (
	ErrorTypeInvalidURL ErrorType = iota
	ErrorTypeAPI
	ErrorTypeFilesystem
	ErrorTypeValidation
	ErrorTypeRateLimit
)

type DownloadError struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *DownloadError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *DownloadError) Unwrap() error {
	return e.Err
}

func (e *DownloadError) Is(target error) bool {
	if t, ok := target.(*DownloadError); ok {
		return e.Type == t.Type
	}
	return false
}

type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, err error, fields ...interface{})
}

type NoOpLogger struct{}

func (l NoOpLogger) Debug(msg string, fields ...interface{})            {}
func (l NoOpLogger) Info(msg string, fields ...interface{})             {}
func (l NoOpLogger) Warn(msg string, fields ...interface{})             {}
func (l NoOpLogger) Error(msg string, err error, fields ...interface{}) {}
