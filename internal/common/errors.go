package common

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrInvalid       = errors.New("invalid input")
	ErrUnavailable   = errors.New("unavailable")
	ErrAlreadyExists = errors.New("already exists")
)

type WrappedError struct {
	Op  string
	Err error
}

func (e *WrappedError) Error() string {
	if e.Op == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *WrappedError) Unwrap() error {
	return e.Err
}

func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return &WrappedError{Op: op, Err: err}
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}
