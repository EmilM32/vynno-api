package domain

import "fmt"

const (
	CodeNotFound             = "not_found"
	CodeInvalidBody          = "invalid_body"
	CodeInvalidQuery         = "invalid_query"
	CodeInvalidJSON          = "invalid_json"
	CodeSessionNotActive     = "session_not_active"
	CodeSessionAlreadyActive = "session_already_active"
	CodeProjectArchived      = "project_archived"
	CodeCodeInUse            = "code_in_use"
	CodeLastActiveProject    = "last_active_project"
	CodeProjectHasSessions   = "project_has_sessions"
	CodeInvalidTransition    = "invalid_transition"
	CodeUnauthorized         = "unauthorized"
	CodeInvalidCredentials   = "invalid_credentials"
	CodeUsernameInUse        = "username_in_use"
)

// Error is a contract error code plus a log/DevTools message.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

func AsError(err error) (*Error, bool) {
	if err == nil {
		return nil, false
	}
	e, ok := err.(*Error)
	return e, ok
}

func ErrNotFound() *Error {
	return NewError(CodeNotFound, "Resource not found.")
}

func ErrInvalidBody(msg string) *Error {
	return NewError(CodeInvalidBody, msg)
}

func ErrInvalidQuery(msg string) *Error {
	return NewError(CodeInvalidQuery, msg)
}

func ErrInvalidJSON() *Error {
	return NewError(CodeInvalidJSON, "Request body is not valid JSON.")
}

func ErrSessionNotActive() *Error {
	return NewError(CodeSessionNotActive, "No active or paused session.")
}

func ErrSessionAlreadyActive() *Error {
	return NewError(CodeSessionAlreadyActive, "An active session already exists. Stop it before starting a new one.")
}

func ErrProjectArchived() *Error {
	return NewError(CodeProjectArchived, "Cannot start a session on an archived project.")
}

func ErrCodeInUse() *Error {
	return NewError(CodeCodeInUse, "Project code is already in use.")
}

func ErrLastActiveProject() *Error {
	return NewError(CodeLastActiveProject, "Cannot archive or delete the last active project.")
}

func ErrProjectHasSessions() *Error {
	return NewError(CodeProjectHasSessions, "Cannot delete a project that has sessions.")
}

func ErrInvalidTransition() *Error {
	return NewError(CodeInvalidTransition, "This action is not valid in the current state.")
}

func ErrUnauthorized() *Error {
	return NewError(CodeUnauthorized, "Authentication required.")
}

func ErrInvalidCredentials() *Error {
	return NewError(CodeInvalidCredentials, "Username or password is incorrect.")
}

func ErrUsernameInUse() *Error {
	return NewError(CodeUsernameInUse, "That username is already in use.")
}
