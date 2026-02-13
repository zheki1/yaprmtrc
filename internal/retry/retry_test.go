package retry

import (
	"context"
	"errors"
	"testing"
)

func TestDoRetry_SuccessOnFirstAttempt(t *testing.T) {
	ctx := context.Background()
	var attempts int

	op := func() error {
		attempts++
		return nil
	}

	isRetryable := func(error) bool {
		return false
	}

	err := DoRetry(ctx, isRetryable, op)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestDoRetry_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	maxAttempts := 2

	op := func() error {
		attempts++
		if attempts < maxAttempts {
			return errors.New("temporary error")
		}
		return nil
	}

	isRetryable := func(err error) bool {
		return err != nil
	}

	err := DoRetry(ctx, isRetryable, op)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if attempts != maxAttempts {
		t.Errorf("expected %d attempts, got %d", maxAttempts, attempts)
	}
}

func TestDoRetry_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	op := func() error {
		attempts++
		return errors.New("non-retryable error")
	}

	isRetryable := func(err error) bool {
		return false
	}

	err := DoRetry(ctx, isRetryable, op)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestDoRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // немедленная отмена

	attempts := 0

	op := func() error {
		attempts++
		return errors.New("some error")
	}

	isRetryable := func(err error) bool {
		return true
	}

	err := DoRetry(ctx, isRetryable, op)

	// Проверяем, что ошибка — это ошибка контекста
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	// DoRetry может не делать попытку, если контекст уже отменён
	if attempts == 0 {
		// Это допустимо: контекст отменён до первой попытки
		return
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestDoRetry_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	op := func() error {
		attempts++
		return errors.New("permanent error")
	}

	isRetryable := func(err error) bool {
		return true
	}

	err := DoRetry(ctx, isRetryable, op)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != len(retryDelays)+1 {
		t.Errorf("expected %d attempts, got %d", len(retryDelays)+1, attempts)
	}
}
