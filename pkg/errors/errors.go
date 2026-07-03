// Package errors defines the application error model used across all layers.
// Every layer returns *Error so the HTTP layer can map kinds to status codes
// without leaking internals.
package errors

import (
	stderrors "errors"
	"fmt"
)

type Kind string

const (
	KindInvalid      Kind = "invalid"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindRateLimited  Kind = "rate_limited"
	KindUnavailable  Kind = "unavailable"
	KindInternal     Kind = "internal"
)

type Error struct {
	Kind    Kind
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

func Wrap(err error, kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message, Err: err}
}

// KindOf extracts the Kind from any error; unknown errors are internal.
func KindOf(err error) Kind {
	var e *Error
	if stderrors.As(err, &e) {
		return e.Kind
	}
	return KindInternal
}

// AsError returns the *Error inside err, or wraps it as internal.
func AsError(err error) *Error {
	var e *Error
	if stderrors.As(err, &e) {
		return e
	}
	return Wrap(err, KindInternal, "internal_error", "internal server error")
}

func Invalid(code, message string) *Error      { return New(KindInvalid, code, message) }
func NotFound(code, message string) *Error     { return New(KindNotFound, code, message) }
func Unauthorized(code, message string) *Error { return New(KindUnauthorized, code, message) }
func Forbidden(code, message string) *Error    { return New(KindForbidden, code, message) }
func Conflict(code, message string) *Error     { return New(KindConflict, code, message) }
func RateLimited(message string) *Error        { return New(KindRateLimited, "rate_limited", message) }
func Internal(err error, message string) *Error {
	return Wrap(err, KindInternal, "internal_error", message)
}
func Unavailable(err error, message string) *Error {
	return Wrap(err, KindUnavailable, "unavailable", message)
}

// Is reports whether err carries the given kind.
func Is(err error, kind Kind) bool { return KindOf(err) == kind }
