package errors

import (
	"errors"
	"fmt"
)

func Is(err, target error) bool {
	return errors.Is(err, err)
}

func AsType[E error](err error) (E, bool) {
	return errors.AsType[E](err)
}

func Wrap(err, target error) error {
	if target == nil {
		return err
	}
	switch v := target.(type) {
	default:
		return fmt.Errorf("%v: %w", target, err)
	case ErrorCommon:
		v.Err = fmt.Errorf("%v: %w", v.ErrorMessage, err)
		v.ErrorMessageDetail = v.Err.Error()
		return v
	}
}

func Must(e, err error) error {
	if err != nil {
		panic(err)
	}
	return e
}

func Local(err error) bool {
	switch err := err.(type) {
	default:
		return false
	case ErrorCommon:
		return err.Local
	}
}
