package services

import (
	"balanca/internal/dto"
	"balanca/internal/repositories"
	"balanca/internal/utils"
	"balanca/pkg/errors"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type UserService interface {
	GetProfile(userID uuid.UUID) (*dto.UserResponse, error)
	UpdateProfile(userID uuid.UUID, req dto.UpdateUserRequest) (*dto.UserResponse, error)
	ChangePassword(userID uuid.UUID, req dto.ChangePasswordRequest) error
	SearchUsers(query string) ([]dto.UserSearchResponse, error)
	GetUserGroups(userID uuid.UUID) ([]dto.GroupResponse, error)
}

type userService struct {
	userRepo  repositories.UserRepository
	groupRepo repositories.GroupRepository
}

func NewUserService(userRepo repositories.UserRepository, groupRepo repositories.GroupRepository) UserService {
	return &userService{
		userRepo:  userRepo,
		groupRepo: groupRepo,
	}
}

func (s *userService) GetProfile(userID uuid.UUID) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, &errors.AppError{Code: "USER_NOT_FOUND", Message: "User not found"}
	}

	return &dto.UserResponse{
		ID:          user.ID,
		PhoneNumber: user.PhoneNumber,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Balance:     user.Balance,
		IsActive:    user.IsActive,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *userService) UpdateProfile(userID uuid.UUID, req dto.UpdateUserRequest) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, &errors.AppError{Code: "USER_NOT_FOUND", Message: "User not found"}
	}

	// Update fields if provided
	if req.Email != "" {
		// Check if email is already taken
		existingUser, _ := s.userRepo.FindByEmail(req.Email)
		if existingUser != nil && existingUser.ID != userID {
			return nil, &errors.AppError{Code: "EMAIL_EXISTS", Message: "Email already taken"}
		}
		user.Email = req.Email
	}

	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}

	if req.LastName != "" {
		user.LastName = req.LastName
	}

	if err := s.userRepo.Update(user); err != nil {
		log.Error().Err(err).Msg("Failed to update user profile")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to update profile"}
	}

	return &dto.UserResponse{
		ID:          user.ID,
		PhoneNumber: user.PhoneNumber,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Balance:     user.Balance,
		IsActive:    user.IsActive,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *userService) ChangePassword(userID uuid.UUID, req dto.ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return &errors.AppError{Code: "USER_NOT_FOUND", Message: "User not found"}
	}

	// Verify old password
	if err := utils.CheckPassword(req.OldPassword, user.PasswordHash); err != nil {
		return &errors.AppError{Code: "INVALID_PASSWORD", Message: "Old password is incorrect"}
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		log.Error().Err(err).Msg("Failed to hash password")
		return &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to change password"}
	}

	user.PasswordHash = hashedPassword
	if err := s.userRepo.Update(user); err != nil {
		log.Error().Err(err).Msg("Failed to update password")
		return &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to change password"}
	}

	return nil
}

func (s *userService) SearchUsers(query string) ([]dto.UserSearchResponse, error) {
	users, err := s.userRepo.SearchByPhoneNumber(query)
	if err != nil {
		log.Error().Err(err).Msg("Failed to search users")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to search users"}
	}

	var response []dto.UserSearchResponse
	for _, user := range users {
		response = append(response, dto.UserSearchResponse{
			ID:          user.ID,
			PhoneNumber: user.PhoneNumber,
			Email:       user.Email,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
		})
	}

	return response, nil
}

func (s *userService) GetUserGroups(userID uuid.UUID) ([]dto.GroupResponse, error) {
	groups, err := s.groupRepo.FindUserGroups(userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user groups")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get groups"}
	}

	var response []dto.GroupResponse
	for _, group := range groups {
		var members []dto.MemberResponse
		for _, member := range group.Members {
			members = append(members, dto.MemberResponse{
				ID:       member.ID,
				UserID:   member.UserID,
				GroupID:  member.GroupID,
				Role:     member.Role,
				Status:   member.Status,
				JoinedAt: member.JoinedAt.Format(time.RFC3339),
				User: dto.UserResponse{
					ID:          member.User.ID,
					PhoneNumber: member.User.PhoneNumber,
					Email:       member.User.Email,
					FirstName:   member.User.FirstName,
					LastName:    member.User.LastName,
					Balance:     member.User.Balance,
					IsActive:    member.User.IsActive,
					CreatedAt:   member.User.CreatedAt.Format(time.RFC3339),
				},
			})
		}

		response = append(response, dto.GroupResponse{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Balance:     group.Balance,
			CreatedBy:   group.CreatedBy,
			IsActive:    group.IsActive,
			CreatedAt:   group.CreatedAt.Format(time.RFC3339),
			Members:     members,
		})
	}

	return response, nil
}
