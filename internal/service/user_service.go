package service

import (
	"context"

	"go-digital-wallet/internal/apperror"
	"go-digital-wallet/internal/dto"
	"go-digital-wallet/internal/model"
	"go-digital-wallet/internal/repository"

	"github.com/google/uuid"
)

// UserService handles user operations.
type UserService struct {
	users      *repository.UserRepository
	pagination paginationConfig
}

// NewUserService constructs a user service.
func NewUserService(users *repository.UserRepository, defaultLimit, maxLimit int) *UserService {
	return &UserService{
		users: users,
		pagination: paginationConfig{
			defaultLimit: defaultLimit,
			maxLimit:     maxLimit,
		},
	}
}

// Create creates a user.
func (s *UserService) Create(ctx context.Context, request dto.CreateUserRequest) (dto.UserResponse, error) {
	user := model.User{
		ExternalRef: request.ExternalRef,
		FullName:    request.FullName,
		Email:       request.Email,
		PhoneNumber: request.PhoneNumber,
		Status:      request.Status,
	}

	if user.Status == "" {
		user.Status = model.UserStatusActive
	}

	if err := s.users.Create(ctx, &user); err != nil {
		if repository.IsDuplicate(err) {
			return dto.UserResponse{}, apperror.Conflict("resource conflicts with existing data")
		}
		return dto.UserResponse{}, apperror.Internal(err)
	}

	return dto.UserFromModel(user), nil
}

// List returns paginated users.
func (s *UserService) List(ctx context.Context, page, limit int) (dto.PageResponse[dto.UserResponse], error) {
	page, limit = normalizePage(page, limit, s.pagination)

	users, total, err := s.users.List(ctx, page, limit)
	if err != nil {
		return dto.PageResponse[dto.UserResponse]{}, apperror.Internal(err)
	}

	items := make([]dto.UserResponse, 0, len(users))
	for _, user := range users {
		items = append(items, dto.UserFromModel(user))
	}

	return responseList(items, page, limit, total), nil
}

// Get fetches a single user.
func (s *UserService) Get(ctx context.Context, id uuid.UUID) (dto.UserResponse, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		if repository.IsNotFound(err) {
			return dto.UserResponse{}, apperror.NotFound("requested resource was not found")
		}
		return dto.UserResponse{}, apperror.Internal(err)
	}

	return dto.UserFromModel(*user), nil
}

// Update updates a user.
func (s *UserService) Update(ctx context.Context, id uuid.UUID, request dto.UpdateUserRequest) (dto.UserResponse, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		if repository.IsNotFound(err) {
			return dto.UserResponse{}, apperror.NotFound("requested resource was not found")
		}
		return dto.UserResponse{}, apperror.Internal(err)
	}

	user.ExternalRef = request.ExternalRef
	user.FullName = request.FullName
	user.Email = request.Email
	user.PhoneNumber = request.PhoneNumber
	user.Status = request.Status

	if err := s.users.Update(ctx, user); err != nil {
		if repository.IsDuplicate(err) {
			return dto.UserResponse{}, apperror.Conflict("resource conflicts with existing data")
		}
		return dto.UserResponse{}, apperror.Internal(err)
	}

	return dto.UserFromModel(*user), nil
}

// Delete soft-deletes a user.
func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.users.GetByID(ctx, id); err != nil {
		if repository.IsNotFound(err) {
			return apperror.NotFound("requested resource was not found")
		}
		return apperror.Internal(err)
	}

	if err := s.users.Delete(ctx, id); err != nil {
		return apperror.Internal(err)
	}

	return nil
}
