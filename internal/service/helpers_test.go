package service_test

import (
	"testing"

	"go-digital-wallet/internal/apperror"
)

func assertAppErrorCode(t *testing.T, err error, wantCode string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected app error %q, got nil", wantCode)
	}

	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != wantCode {
		t.Fatalf("expected app error code %q, got %q", wantCode, appErr.Code)
	}
}
