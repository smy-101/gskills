package initializer

import "fmt"

type ErrorType int

const (
	ErrTypeBinaryCopy ErrorType = iota
	ErrTypeDirCreate
	ErrTypeConfigWrite
	ErrTypeShellDetection
	ErrTypePathResolution
)

type InitError struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *InitError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *InitError) Unwrap() error {
	return e.Err
}

func (e *InitError) Is(target error) bool {
	t, ok := target.(*InitError)
	if !ok {
		return false
	}
	return e.Type == t.Type
}
