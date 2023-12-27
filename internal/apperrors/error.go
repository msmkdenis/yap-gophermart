package apperrors

import (
	"errors"
	"fmt"
)

var (
	ErrLoginAlreadyExists              = errors.New("login already exists")
	ErrUserNotFound                    = errors.New("user not found")
	ErrInvalidPassword                 = errors.New("invalid password")
	ErrUnableToGetUserLoginFromContext = errors.New("unable to get user login from context")
	ErrEmptyOrderRequest               = errors.New("empty order")
	ErrOrderUploadedByAnotherUser      = errors.New("order uploaded by another user")
	ErrOrderUploadedByUser             = errors.New("order uploaded by User")
	ErrOrderNotFound                   = errors.New("order not found")
	ErrBadNumber                       = errors.New("bad number")
	ErrNoOrders                        = errors.New("no orders")
	ErrRateLimit                       = errors.New("rate limit")
	ErrBalanceNotFound                 = errors.New("balance not found")
	ErrInsufficientFunds               = errors.New("insufficient funds")
	ErrNoWithdrawals                   = errors.New("no withdrawals")
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