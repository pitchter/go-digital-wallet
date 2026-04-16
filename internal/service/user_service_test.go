package service_test

import (
	"context"
	"testing"

	"go-digital-wallet/internal/dto"
	"go-digital-wallet/internal/repository"
	"go-digital-wallet/internal/testutil"

	"github.com/google/uuid"
)

func TestUserServiceCreateSuccessDefaultsActive(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)

	response, err := env.UserService.Create(context.Background(), dto.CreateUserRequest{
		FullName:    "Alice Example",
		Email:       "alice@example.com",
		PhoneNumber: "0800000001",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if response.ID == uuid.Nil {
		t.Fatal("expected generated user ID")
	}
	if response.Status != "active" {
		t.Fatalf("expected default status active, got %s", response.Status)
	}
}

func TestUserServiceCreateConflict(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	ctx := context.Background()

	request := dto.CreateUserRequest{
		FullName:    "Alice",
		Email:       "alice@example.com",
		PhoneNumber: "0800000001",
	}

	if _, err := env.UserService.Create(ctx, request); err != nil {
		t.Fatalf("create user: %v", err)
	}

	_, err := env.UserService.Create(ctx, dto.CreateUserRequest{
		FullName:    "Alice Clone",
		Email:       "alice@example.com",
		PhoneNumber: "0800000002",
	})
	assertAppErrorCode(t, err, "resource_conflict")
}

func TestUserServiceGetSuccess(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "get-user@example.com")

	response, err := env.UserService.Get(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	if response.ID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, response.ID)
	}
	if response.Email != user.Email {
		t.Fatalf("expected email %s, got %s", user.Email, response.Email)
	}
}

func TestUserServiceUpdateSuccess(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "update-user@example.com")
	externalRef := "updated-ref"

	response, err := env.UserService.Update(context.Background(), user.ID, dto.UpdateUserRequest{
		ExternalRef: &externalRef,
		FullName:    "Updated User",
		Email:       "updated-user@example.com",
		PhoneNumber: "0800009999",
		Status:      "inactive",
	})
	if err != nil {
		t.Fatalf("update user: %v", err)
	}

	if response.FullName != "Updated User" {
		t.Fatalf("expected updated full name, got %s", response.FullName)
	}
	if response.Status != "inactive" {
		t.Fatalf("expected updated status inactive, got %s", response.Status)
	}

	stored, err := env.UserRepo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("load updated user: %v", err)
	}
	if stored.Email != "updated-user@example.com" {
		t.Fatalf("expected updated email, got %s", stored.Email)
	}
}

func TestUserServiceDeleteSuccess(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "delete-user@example.com")

	if err := env.UserService.Delete(context.Background(), user.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	_, err := env.UserRepo.GetByID(context.Background(), user.ID)
	if !repository.IsNotFound(err) {
		t.Fatalf("expected repository not found after delete, got %v", err)
	}
}

func TestUserServiceMissingResourceErrors(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	missingID := uuid.New()

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "get missing user",
			run: func() error {
				_, err := env.UserService.Get(context.Background(), missingID)
				return err
			},
		},
		{
			name: "update missing user",
			run: func() error {
				_, err := env.UserService.Update(context.Background(), missingID, dto.UpdateUserRequest{
					FullName:    "Missing User",
					Email:       "missing@example.com",
					PhoneNumber: "0800011111",
					Status:      "active",
				})
				return err
			},
		},
		{
			name: "delete missing user",
			run: func() error {
				return env.UserService.Delete(context.Background(), missingID)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertAppErrorCode(t, tt.run(), "resource_not_found")
		})
	}
}
