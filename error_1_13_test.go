package errors

import (
	"io"
	"testing"
)

func TestIs113(t *testing.T) {
	t.Parallel()
	custErr := errorWithCustomIs{
		Key: "TestForFun",
		Err: io.EOF,
	}

	shouldMatch := errorWithCustomIs{
		Key: "TestForFun",
	}

	shouldNotMatch := errorWithCustomIs{Key: "notOk"}

	if !Is(custErr, shouldMatch) {
		t.Errorf("custErr is not a TestForFun customError")
	}

	if Is(custErr, shouldNotMatch) {
		t.Errorf("custErr is a notOk customError")
	}

	if !Is(custErr, Wrap(shouldMatch, 0)) {
		t.Errorf("custErr is not a Wrap(TestForFun customError)")
	}

	if Is(custErr, Wrap(shouldNotMatch, 0)) {
		t.Errorf("custErr is a Wrap(notOk customError)")
	}

	if !Is(Wrap(custErr, 0), shouldMatch) {
		t.Errorf("Wrap(custErr) is not a TestForFun customError")
	}

	if Is(Wrap(custErr, 0), shouldNotMatch) {
		t.Errorf("Wrap(custErr) is a notOk customError")
	}

	if !Is(Wrap(custErr, 0), Wrap(shouldMatch, 0)) {
		t.Errorf("Wrap(custErr) is not a Wrap(TestForFun customError)")
	}

	if Is(Wrap(custErr, 0), Wrap(shouldNotMatch, 0)) {
		t.Errorf("Wrap(custErr) is a Wrap(notOk customError)")
	}
}

type errorWithCustomIs struct {
	Key string
	Err error
}

func (ewci errorWithCustomIs) Error() string {
	return "[" + ewci.Key + "]: " + ewci.Err.Error()
}

func (ewci errorWithCustomIs) Is(target error) bool {
	matched, ok := target.(errorWithCustomIs)
	return ok && matched.Key == ewci.Key
}
