// Package assert provides minimal generic test assertions on the standard
// testing package, replacing the third-party testify dependency. All
// assertions are fatal (the test stops on first failure).
package assert

import (
	"cmp"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// suffix renders optional trailing message-and-args the way testify did.
func suffix(msgAndArgs []any) string {
	if len(msgAndArgs) == 0 {
		return ""
	}
	if format, ok := msgAndArgs[0].(string); ok && len(msgAndArgs) > 1 {
		return ": " + fmt.Sprintf(format, msgAndArgs[1:]...)
	}
	return ": " + fmt.Sprint(msgAndArgs...)
}

// Equal asserts got == want.
func Equal[T comparable](t testing.TB, got, want T, msgAndArgs ...any) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v%s", got, want, suffix(msgAndArgs))
	}
}

// NotEqual asserts got != want.
func NotEqual[T comparable](t testing.TB, got, want T, msgAndArgs ...any) {
	t.Helper()
	if got == want {
		t.Fatalf("got %v, want anything else%s", got, suffix(msgAndArgs))
	}
}

// DeepEqual asserts reflect.DeepEqual(got, want), for values that are not
// comparable with == (slices, maps, structs containing them).
func DeepEqual(t testing.TB, got, want any, msgAndArgs ...any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v%s", got, want, suffix(msgAndArgs))
	}
}

// NoError asserts err is nil.
func NoError(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v%s", err, suffix(msgAndArgs))
	}
}

// Error asserts err is non-nil.
func Error(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error, got nil%s", suffix(msgAndArgs))
	}
}

// isNil reports whether v is nil, including typed-nil pointers, maps, slices,
// channels, funcs, and interfaces.
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

// Nil asserts v is nil.
func Nil(t testing.TB, v any, msgAndArgs ...any) {
	t.Helper()
	if !isNil(v) {
		t.Fatalf("got %v, want nil%s", v, suffix(msgAndArgs))
	}
}

// NotNil asserts v is non-nil.
func NotNil(t testing.TB, v any, msgAndArgs ...any) {
	t.Helper()
	if isNil(v) {
		t.Fatalf("got nil, want non-nil%s", suffix(msgAndArgs))
	}
}

// True asserts v is true.
func True(t testing.TB, v bool, msgAndArgs ...any) {
	t.Helper()
	if !v {
		t.Fatalf("got false, want true%s", suffix(msgAndArgs))
	}
}

// False asserts v is false.
func False(t testing.TB, v bool, msgAndArgs ...any) {
	t.Helper()
	if v {
		t.Fatalf("got true, want false%s", suffix(msgAndArgs))
	}
}

// Contains asserts s contains substr.
func Contains(t testing.TB, s, substr string, msgAndArgs ...any) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("%q does not contain %q%s", s, substr, suffix(msgAndArgs))
	}
}

// NotContains asserts s does not contain substr.
func NotContains(t testing.TB, s, substr string, msgAndArgs ...any) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Fatalf("%q contains %q%s", s, substr, suffix(msgAndArgs))
	}
}

// Empty asserts s is the empty string.
func Empty(t testing.TB, s string, msgAndArgs ...any) {
	t.Helper()
	if s != "" {
		t.Fatalf("got %q, want empty%s", s, suffix(msgAndArgs))
	}
}

// NotEmpty asserts s is a non-empty string.
func NotEmpty(t testing.TB, s string, msgAndArgs ...any) {
	t.Helper()
	if s == "" {
		t.Fatalf("got empty string, want non-empty%s", suffix(msgAndArgs))
	}
}

// Len asserts the slice has exactly want elements.
func Len[T any](t testing.TB, got []T, want int, msgAndArgs ...any) {
	t.Helper()
	if len(got) != want {
		t.Fatalf("got len %d, want %d%s", len(got), want, suffix(msgAndArgs))
	}
}

// GreaterOrEqual asserts a >= b.
func GreaterOrEqual[T cmp.Ordered](t testing.TB, a, b T, msgAndArgs ...any) {
	t.Helper()
	if a < b {
		t.Fatalf("%v is not greater than or equal to %v%s", a, b, suffix(msgAndArgs))
	}
}

// NotPanics asserts fn returns without panicking.
func NotPanics(t testing.TB, fn func(), msgAndArgs ...any) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v%s", r, suffix(msgAndArgs))
		}
	}()
	fn()
}
