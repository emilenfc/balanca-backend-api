package services

import (
	"balanca/internal/dto"
	"balanca/internal/models"
	"balanca/internal/repositories"
	"balanca/internal/utils"
	"balanca/pkg/errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type AuthService interface {
	Register(req dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(req dto.LoginRequest) (*dto.AuthResponse, error)
	RefreshToken(refreshToken string) (*dto.AuthResponse, error)
	Logout(userID uuid.UUID) error
}

type authService struct {
	userRepo repositories.UserRepository
	config   struct {
		jwtSecret              string
		jwtExpiration          time.Duration
		refreshTokenExpiration time.Duration
	}
}

func NewAuthService(userRepo repositories.UserRepository, jwtSecret string, jwtExp, refreshExp time.Duration) AuthService {
	return &authService{
		userRepo: userRepo,
		config: struct {
			jwtSecret              string
			jwtExpiration          time.Duration
			refreshTokenExpiration time.Duration
		}{
			jwtSecret:              jwtSecret,
			jwtExpiration:          jwtExp,
			refreshTokenExpiration: refreshExp,
		},
	}
}

func (s *authService) Register(req dto.RegisterRequest) (*dto.AuthResponse, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.FindByPhoneNumber(req.PhoneNumber)
	if existingUser != nil {
		fmt.Println("\nexistingUser | ",existingUser)
		return nil, &errors.AppError{Code: "USER_EXISTS", Message: "User with this phone number already exists"}
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		log.Error().Err(err).Msg("Failed to hash password")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to process request"}
	}

	// Create user
	user := &models.User{
		PhoneNumber:  req.PhoneNumber,
		Email:        req.Email,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		PasswordHash: hashedPassword,
		Balance:      0,
		IsActive:     true,
	}

	if err := s.userRepo.Create(user); err != nil {
		log.Error().Err(err).Msg("Failed to create user")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create user"}
	}

	// Generate tokens
	accessToken, err := utils.GenerateAccessToken(
		user.ID,
		user.PhoneNumber,
		user.Email,
		s.config.jwtSecret,
		s.config.jwtExpiration,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate access token")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate token"}
	}

	refreshToken, err := utils.GenerateRefreshToken(
		user.ID,
		s.config.jwtSecret,
		s.config.refreshTokenExpiration,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate refresh token")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate token"}
	}

	// Create response
	response := &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: dto.UserResponse{
			ID:          user.ID,
			PhoneNumber: user.PhoneNumber,
			Email:       user.Email,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
			Balance:     user.Balance,
			IsActive:    user.IsActive,
			CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		},
	}

	return response, nil
}

func (s *authService) Login(req dto.LoginRequest) (*dto.AuthResponse, error) {
	// Find user by phone number
	user, err := s.userRepo.FindByPhoneNumber(req.PhoneNumber)
	if err != nil {
		log.Error().Err(err).Msg("User not found")
		return nil, &errors.AppError{Code: "INVALID_CREDENTIALS", Message: "Invalid phone number or password"}
	}

	// Check password
	if err := utils.CheckPassword(req.Password, user.PasswordHash); err != nil {
		log.Error().Err(err).Msg("Invalid password")
		return nil, &errors.AppError{Code: "INVALID_CREDENTIALS", Message: "Invalid phone number or password"}
	}

	// Check if user is active
	if !user.IsActive {
		return nil, &errors.AppError{Code: "USER_INACTIVE", Message: "Account is inactive"}
	}

	// Generate tokens
	accessToken, err := utils.GenerateAccessToken(
		user.ID,
		user.PhoneNumber,
		user.Email,
		s.config.jwtSecret,
		s.config.jwtExpiration,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate access token")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate token"}
	}

	refreshToken, err := utils.GenerateRefreshToken(
		user.ID,
		s.config.jwtSecret,
		s.config.refreshTokenExpiration,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate refresh token")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate token"}
	}

	// Create response
	response := &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: dto.UserResponse{
			ID:          user.ID,
			PhoneNumber: user.PhoneNumber,
			Email:       user.Email,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
			Balance:     user.Balance,
			IsActive:    user.IsActive,
			CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		},
	}

	return response, nil
}

func (s *authService) RefreshToken(refreshToken string) (*dto.AuthResponse, error) {
	// Validate refresh token
	claims, err := utils.ValidateToken(refreshToken, s.config.jwtSecret)
	if err != nil {
		return nil, &errors.AppError{Code: "INVALID_TOKEN", Message: "Invalid refresh token"}
	}

	// Find user
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, &errors.AppError{Code: "USER_NOT_FOUND", Message: "User not found"}
	}

	// Generate new tokens
	accessToken, err := utils.GenerateAccessToken(
		user.ID,
		user.PhoneNumber,
		user.Email,
		s.config.jwtSecret,
		s.config.jwtExpiration,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate access token")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate token"}
	}

	newRefreshToken, err := utils.GenerateRefreshToken(
		user.ID,
		s.config.jwtSecret,
		s.config.refreshTokenExpiration,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate refresh token")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate token"}
	}

	// Create response
	response := &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		User: dto.UserResponse{
			ID:          user.ID,
			PhoneNumber: user.PhoneNumber,
			Email:       user.Email,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
			Balance:     user.Balance,
			IsActive:    user.IsActive,
			CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		},
	}

	return response, nil
}

func (s *authService) Logout(userID uuid.UUID) error {
	// In a real application, you would:
	// 1. Add the refresh token to a blacklist
	// 2. Clear any session data
	// For now, we just return success
	return nil
}
