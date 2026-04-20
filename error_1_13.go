package errors

import (
	baseErrors "errors"
)

// As finds the first error in err's tree that matches target, and if one is found, sets
// target to that error value and returns true. Otherwise, it returns false.
//
// For more information see stdlib errors.As.
func As(err error, target any) bool {
	return baseErrors.As(err, target)
}

// Is detects whether the error is equal to a given error. Errors
// are considered equal by this function if they are matched by errors.Is
// or if their contained errors are matched through errors.Is.
func Is(err error, original error) bool {
	if baseErrors.Is(err, original) {
		return true
	}
	if original, ok := original.(*Error); ok {
		return Is(err, original.Err)
	}
	return false
}
