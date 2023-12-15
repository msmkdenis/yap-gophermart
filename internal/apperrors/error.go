package apperrors

import (
	"errors"
	"fmt"
)

var (
	ErrLoginAlreadyExists = errors.New("login already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidPassword    = errors.New("invalid password")
)

type ValueError struct {
	caller  string
	message string
	err     error
}

func NewValueError(message string, caller string, err error) error {
	return &ValueError{
		caller:  caller,
		message: message,
		err:     err,
	}
}

func (v *ValueError) Error() string {
	return fmt.Sprintf("%s %s %s", v.caller, v.message, v.err)
}

func (v *ValueError) Unwrap() error {
	return v.err
}
