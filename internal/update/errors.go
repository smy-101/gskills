package update

import "fmt"

type UpdateErrorType int

const (
	UpdateErrorTypeCheck UpdateErrorType = iota
	UpdateErrorTypeDownload
	UpdateErrorTypeRegistry
	UpdateErrorTypeNotFound
)

type UpdateError struct {
	Type    UpdateErrorType
	Message string
	Err     error
	Skill   string
}

func (e *UpdateError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s '%s': %v", e.Message, e.Skill, e.Err)
	}
	if e.Skill != "" {
		return fmt.Sprintf("%s '%s'", e.Message, e.Skill)
	}
	return e.Message
}

func (e *UpdateError) Unwrap() error {
	return e.Err
}

func (e *UpdateError) Is(target error) bool {
	if t, ok := target.(*UpdateError); ok {
		return e.Type == t.Type
	}
	return false
}
