package turborpc

import (
	"errors"
	"fmt"
	"testing"
)

func assertEqual[T comparable](t *testing.T, a T, b T, msgAndArgs ...any) bool {
	if a != b {
		t.Fatalf(messageFromMsgAndArgs(msgAndArgs...))
	}

	return true
}

func assertNoError(t *testing.T, err error, msgAndArgs ...any) bool {
	if err != nil {
		t.Fatalf(messageFromMsgAndArgs(msgAndArgs...))
	}

	return true
}

func assertErrorIs(t *testing.T, expected error, value error, msgAndArgs ...any) bool {
	if !errors.Is(value, expected) {
		t.Fatalf(messageFromMsgAndArgs(msgAndArgs...))
	}

	return true
}

func messageFromMsgAndArgs(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 {
		return ""
	}

	if len(msgAndArgs) == 1 {
		msg := msgAndArgs[0]

		if msgAsStr, ok := msg.(string); ok {
			return msgAsStr
		}

		return fmt.Sprintf("%+v", msg)
	}

	if len(msgAndArgs) > 1 {
		return fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}

	return ""
}
