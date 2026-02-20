package errors

import (
	"errors"
	"testing"
)

func TestDomainError(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		err := New(CodeNotFound, "resource not found")
		if err.Error() != "[NOT_FOUND] resource not found" {
			t.Errorf("expected [NOT_FOUND] resource not found, got %s", err.Error())
		}
	})

	t.Run("Wrap", func(t *testing.T) {
		original := errors.New("original error")
		err := Wrap(original, CodeInternal, "internal failure")
		expected := "[INTERNAL_ERROR] internal failure: original error"
		if err.Error() != expected {
			t.Errorf("expected %s, got %s", expected, err.Error())
		}
	})

	t.Run("IsCode", func(t *testing.T) {
		err := New(CodeValidationError, "invalid input")
		if !IsCode(err, CodeValidationError) {
			t.Error("expected IsCode to return true for CodeValidationError")
		}
		if IsCode(err, CodeNotFound) {
			t.Error("expected IsCode to return false for CodeNotFound")
		}
	})

	t.Run("IsCodeWithWrapped", func(t *testing.T) {
		original := errors.New("original error")
		err := Wrap(original, CodeInternal, "internal failure")
		if !IsCode(err, CodeInternal) {
			t.Error("expected IsCode to return true for wrapped CodeInternal")
		}
	})
}
