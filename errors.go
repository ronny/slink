package slink

import "fmt"

type ErrInvalidLinkURL struct {
	msg string
}

func (e *ErrInvalidLinkURL) Error() string {
	return fmt.Sprintf("ErrInvalidLinkURL: %s", e.msg)
}

type ErrInvalidShortLinkID struct {
	msg string
}

func (e *ErrInvalidShortLinkID) Error() string {
	return fmt.Sprintf("ErrInvalidShortLinkID: %s", e.msg)
}

type ErrCreateAttemptsExhausted struct {
	attempts int
}

func (e *ErrCreateAttemptsExhausted) Error() string {
	return fmt.Sprintf("ErrCreateAttemptsExhausted: failed to create after %d attempts", e.attempts)
}
