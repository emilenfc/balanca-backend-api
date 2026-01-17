package services

import (
	"balanca/internal/dto"
	"balanca/internal/models"
	"balanca/internal/repositories"
	"balanca/pkg/errors"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type PlannedExpenseService interface {
	CreatePersonalExpense(userID uuid.UUID, req dto.CreatePlannedExpenseRequest) (*dto.PlannedExpenseResponse, error)
	CreateGroupExpense(userID uuid.UUID, req dto.CreatePlannedExpenseRequest) (*dto.PlannedExpenseResponse, error)
	GetPersonalExpenses(userID uuid.UUID, status string, page, limit int) ([]dto.PlannedExpenseResponse, int64, error)
	GetGroupExpenses(userID, groupID uuid.UUID, status string, page, limit int) ([]dto.PlannedExpenseResponse, int64, error)
	GetExpense(userID, expenseID uuid.UUID) (*dto.PlannedExpenseResponse, error)
	UpdateExpense(userID, expenseID uuid.UUID, req dto.UpdatePlannedExpenseRequest) (*dto.PlannedExpenseResponse, error)
	DeleteExpense(userID, expenseID uuid.UUID) error
	MarkAsBought(userID, expenseID uuid.UUID, req dto.MarkAsBoughtRequest) (*dto.PlannedExpenseResponse, error)
	MarkAsCancelled(userID, expenseID uuid.UUID) error
	GetOverdueExpenses(userID uuid.UUID) ([]dto.PlannedExpenseResponse, error)
}

type plannedExpenseService struct {
	expenseRepo repositories.PlannedExpenseRepository
	userRepo    repositories.UserRepository
	groupRepo   repositories.GroupRepository
	auditRepo   repositories.AuditLogRepository
	db          *gorm.DB
}

func NewPlannedExpenseService(
	expenseRepo repositories.PlannedExpenseRepository,
	userRepo repositories.UserRepository,
	groupRepo repositories.GroupRepository,
	auditRepo repositories.AuditLogRepository,
	db *gorm.DB,
) PlannedExpenseService {
	return &plannedExpenseService{
		expenseRepo: expenseRepo,
		userRepo:    userRepo,
		groupRepo:   groupRepo,
		auditRepo:   auditRepo,
		db:          db,
	}
}

func (s *plannedExpenseService) CreatePersonalExpense(userID uuid.UUID, req dto.CreatePlannedExpenseRequest) (*dto.PlannedExpenseResponse, error) {
	expense := &models.PlannedExpense{
		Item:           req.Item,
		Description:    req.Description,
		EstimatedPrice: req.EstimatedPrice,
		Category:       req.Category,
		Priority:       req.Priority,
		Status:         "planned",
		UserID:         userID,
		DueDate:        req.DueDate,
	}

	if err := s.expenseRepo.Create(expense); err != nil {
		log.Error().Err(err).Msg("Failed to create planned expense")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create planned expense"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "planned_expense",
		EntityID:    expense.ID,
		Action:      "create",
		Changes:     map[string]interface{}{"item": req.Item, "estimated_price": req.EstimatedPrice},
		PerformedBy: userID,
	}

	if err := s.auditRepo.Create(auditLog); err != nil {
		log.Error().Err(err).Msg("Failed to create audit log")
	}

	// Get full expense data
	fullExpense, err := s.expenseRepo.FindByID(expense.ID)
	if err != nil {
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get expense data"}
	}

	return s.mapExpenseToResponse(fullExpense), nil
}

func (s *plannedExpenseService) CreateGroupExpense(userID uuid.UUID, req dto.CreatePlannedExpenseRequest) (*dto.PlannedExpenseResponse, error) {
	if req.GroupID == nil {
		return nil, &errors.AppError{Code: "INVALID_REQUEST", Message: "Group ID is required"}
	}

	// Check if user is a member of the group
	userGroup, err := s.groupRepo.FindByUserAndGroup(userID, *req.GroupID)
	if err != nil || userGroup.Status != "active" {
		return nil, &errors.AppError{Code: "FORBIDDEN", Message: "You are not a member of this group"}
	}

	expense := &models.PlannedExpense{
		Item:           req.Item,
		Description:    req.Description,
		EstimatedPrice: req.EstimatedPrice,
		Category:       req.Category,
		Priority:       req.Priority,
		Status:         "planned",
		UserID:         userID,
		GroupID:        req.GroupID,
		DueDate:        req.DueDate,
	}

	if err := s.expenseRepo.Create(expense); err != nil {
		log.Error().Err(err).Msg("Failed to create planned expense")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create planned expense"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "planned_expense",
		EntityID:    expense.ID,
		Action:      "create",
		Changes:     map[string]interface{}{"item": req.Item, "estimated_price": req.EstimatedPrice},
		PerformedBy: userID,
		GroupID:     req.GroupID,
	}

	if err := s.auditRepo.Create(auditLog); err != nil {
		log.Error().Err(err).Msg("Failed to create audit log")
	}

	// Get full expense data
	fullExpense, err := s.expenseRepo.FindByID(expense.ID)
	if err != nil {
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get expense data"}
	}

	return s.mapExpenseToResponse(fullExpense), nil
}

func (s *plannedExpenseService) GetPersonalExpenses(userID uuid.UUID, status string, page, limit int) ([]dto.PlannedExpenseResponse, int64, error) {
	expenses, total, err := s.expenseRepo.FindByUser(userID, status, page, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get personal expenses")
		return nil, 0, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get expenses"}
	}

	var response []dto.PlannedExpenseResponse
	for _, expense := range expenses {
		response = append(response, *s.mapExpenseToResponse(&expense))
	}

	return response, total, nil
}

func (s *plannedExpenseService) GetGroupExpenses(userID, groupID uuid.UUID, status string, page, limit int) ([]dto.PlannedExpenseResponse, int64, error) {
	// Check if user is a member of the group
	userGroup, err := s.groupRepo.FindByUserAndGroup(userID, groupID)
	if err != nil || userGroup.Status != "active" {
		return nil, 0, &errors.AppError{Code: "FORBIDDEN", Message: "You are not a member of this group"}
	}

	expenses, total, err := s.expenseRepo.FindByGroup(groupID, status, page, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get group expenses")
		return nil, 0, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get expenses"}
	}

	var response []dto.PlannedExpenseResponse
	for _, expense := range expenses {
		response = append(response, *s.mapExpenseToResponse(&expense))
	}

	return response, total, nil
}

func (s *plannedExpenseService) GetExpense(userID, expenseID uuid.UUID) (*dto.PlannedExpenseResponse, error) {
	expense, err := s.expenseRepo.FindByID(expenseID)
	if err != nil {
		return nil, &errors.AppError{Code: "EXPENSE_NOT_FOUND", Message: "Expense not found"}
	}

	// Check if user has access to this expense
	if expense.UserID != userID {
		if expense.GroupID != nil {
			// Check if user is a member of the group
			userGroup, err := s.groupRepo.FindByUserAndGroup(userID, *expense.GroupID)
			if err != nil || userGroup.Status != "active" {
				return nil, &errors.AppError{Code: "FORBIDDEN", Message: "Access denied"}
			}
		} else {
			return nil, &errors.AppError{Code: "FORBIDDEN", Message: "Access denied"}
		}
	}

	return s.mapExpenseToResponse(expense), nil
}

func (s *plannedExpenseService) UpdateExpense(userID, expenseID uuid.UUID, req dto.UpdatePlannedExpenseRequest) (*dto.PlannedExpenseResponse, error) {
	expense, err := s.expenseRepo.FindByID(expenseID)
	if err != nil {
		return nil, &errors.AppError{Code: "EXPENSE_NOT_FOUND", Message: "Expense not found"}
	}

	// Check if user has permission to update
	if expense.UserID != userID {
		if expense.GroupID != nil {
			// For group expenses, check if user is a member
			userGroup, err := s.groupRepo.FindByUserAndGroup(userID, *expense.GroupID)
			if err != nil || userGroup.Status != "active" {
				return nil, &errors.AppError{Code: "FORBIDDEN", Message: "Access denied"}
			}
		} else {
			return nil, &errors.AppError{Code: "FORBIDDEN", Message: "Access denied"}
		}
	}

	// Record changes for audit log
	changes := make(map[string]interface{})

	// Update fields if provided
	if req.Item != nil && *req.Item != expense.Item {
		changes["item"] = map[string]interface{}{"old": expense.Item, "new": *req.Item}
		expense.Item = *req.Item
	}

	if req.Description != nil && *req.Description != expense.Description {
		changes["description"] = map[string]interface{}{"old": expense.Description, "new": *req.Description}
		expense.Description = *req.Description
	}

	if req.EstimatedPrice != nil && *req.EstimatedPrice != expense.EstimatedPrice {
		changes["estimated_price"] = map[string]interface{}{"old": expense.EstimatedPrice, "new": *req.EstimatedPrice}
		expense.EstimatedPrice = *req.EstimatedPrice
	}

	if req.Category != nil && *req.Category != expense.Category {
		changes["category"] = map[string]interface{}{"old": expense.Category, "new": *req.Category}
		expense.Category = *req.Category
	}

	if req.Priority != nil && *req.Priority != expense.Priority {
		changes["priority"] = map[string]interface{}{"old": expense.Priority, "new": *req.Priority}
		expense.Priority = *req.Priority
	}

	if req.DueDate != nil {
		oldDueDate := expense.DueDate
		if oldDueDate != nil && req.DueDate != nil && !oldDueDate.Equal(*req.DueDate) {
			changes["due_date"] = map[string]interface{}{"old": oldDueDate, "new": req.DueDate}
		} else if oldDueDate == nil && req.DueDate != nil {
			changes["due_date"] = map[string]interface{}{"old": nil, "new": req.DueDate}
		}
		expense.DueDate = req.DueDate
	}

	if err := s.expenseRepo.Update(expense); err != nil {
		log.Error().Err(err).Msg("Failed to update expense")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to update expense"}
	}

	// Create audit log if there were changes
	if len(changes) > 0 {
		auditLog := &models.AuditLog{
			Entity:      "planned_expense",
			EntityID:    expense.ID,
			Action:      "update",
			Changes:     changes,
			PerformedBy: userID,
			GroupID:     expense.GroupID,
		}

		if err := s.auditRepo.Create(auditLog); err != nil {
			log.Error().Err(err).Msg("Failed to create audit log")
		}
	}

	// Get updated expense data
	updatedExpense, err := s.expenseRepo.FindByID(expenseID)
	if err != nil {
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get updated expense data"}
	}

	return s.mapExpenseToResponse(updatedExpense), nil
}

func (s *plannedExpenseService) DeleteExpense(userID, expenseID uuid.UUID) error {
	expense, err := s.expenseRepo.FindByID(expenseID)
	if err != nil {
		return &errors.AppError{Code: "EXPENSE_NOT_FOUND", Message: "Expense not found"}
	}

	// Check if user has permission to delete
	if expense.UserID != userID {
		if expense.GroupID != nil {
			// For group expenses, check if user is a manager
			userGroup, err := s.groupRepo.FindByUserAndGroup(userID, *expense.GroupID)
			if err != nil || userGroup.Status != "active" || userGroup.Role != "manager" {
				return &errors.AppError{Code: "FORBIDDEN", Message: "Only managers can delete group expenses"}
			}
		} else {
			return &errors.AppError{Code: "FORBIDDEN", Message: "Access denied"}
		}
	}

	if err := s.expenseRepo.Delete(expenseID); err != nil {
		log.Error().Err(err).Msg("Failed to delete expense")
		return &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to delete expense"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "planned_expense",
		EntityID:    expenseID,
		Action:      "delete",
		PerformedBy: userID,
		GroupID:     expense.GroupID,
	}

	if err := s.auditRepo.Create(auditLog); err != nil {
		log.Error().Err(err).Msg("Failed to create audit log")
	}

	return nil
}

func (s *plannedExpenseService) MarkAsBought(userID, expenseID uuid.UUID, req dto.MarkAsBoughtRequest) (*dto.PlannedExpenseResponse, error) {
	expense, err := s.expenseRepo.FindByID(expenseID)
	if err != nil {
		return nil, &errors.AppError{Code: "EXPENSE_NOT_FOUND", Message: "Expense not found"}
	}

	// Check if expense is in planned status
	if expense.Status != "planned" {
		return nil, &errors.AppError{Code: "INVALID_STATUS", Message: "Expense is not in planned status"}
	}

	// For personal expenses, just mark as bought
	if expense.GroupID == nil {
		if expense.UserID != userID {
			return nil, &errors.AppError{Code: "FORBIDDEN", Message: "Access denied"}
		}

		if err := s.expenseRepo.MarkAsBought(expenseID, req.ActualPrice, userID); err != nil {
			log.Error().Err(err).Msg("Failed to mark expense as bought")
			return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to mark expense as bought"}
		}

		// Create audit log
		auditLog := &models.AuditLog{
			Entity:      "planned_expense",
			EntityID:    expenseID,
			Action:      "mark_as_bought",
			Changes:     map[string]interface{}{"actual_price": req.ActualPrice},
			PerformedBy: userID,
		}

		if err := s.auditRepo.Create(auditLog); err != nil {
			log.Error().Err(err).Msg("Failed to create audit log")
		}
	} else {
		// For group expenses, use the transaction service to handle payment
		// This will be called from the group transaction flow
		return nil, &errors.AppError{Code: "USE_TRANSACTION_FLOW", Message: "Use group transaction flow to pay for group expenses"}
	}

	// Get updated expense data
	updatedExpense, err := s.expenseRepo.FindByID(expenseID)
	if err != nil {
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get updated expense data"}
	}

	return s.mapExpenseToResponse(updatedExpense), nil
}

func (s *plannedExpenseService) MarkAsCancelled(userID, expenseID uuid.UUID) error {
	expense, err := s.expenseRepo.FindByID(expenseID)
	if err != nil {
		return &errors.AppError{Code: "EXPENSE_NOT_FOUND", Message: "Expense not found"}
	}

	// Check if user has permission
	if expense.UserID != userID {
		if expense.GroupID != nil {
			// For group expenses, check if user is a member
			userGroup, err := s.groupRepo.FindByUserAndGroup(userID, *expense.GroupID)
			if err != nil || userGroup.Status != "active" {
				return &errors.AppError{Code: "FORBIDDEN", Message: "Access denied"}
			}
		} else {
			return &errors.AppError{Code: "FORBIDDEN", Message: "Access denied"}
		}
	}

	if err := s.expenseRepo.MarkAsCancelled(expenseID); err != nil {
		log.Error().Err(err).Msg("Failed to mark expense as cancelled")
		return &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to mark expense as cancelled"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "planned_expense",
		EntityID:    expenseID,
		Action:      "mark_as_cancelled",
		PerformedBy: userID,
		GroupID:     expense.GroupID,
	}

	if err := s.auditRepo.Create(auditLog); err != nil {
		log.Error().Err(err).Msg("Failed to create audit log")
	}

	return nil
}

func (s *plannedExpenseService) GetOverdueExpenses(userID uuid.UUID) ([]dto.PlannedExpenseResponse, error) {
	// Get user's personal overdue expenses
	personalExpenses, err := s.expenseRepo.FindOverdue(0) // 0 days means all past due
	if err != nil {
		log.Error().Err(err).Msg("Failed to get overdue expenses")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get overdue expenses"}
	}

	// Filter only user's expenses
	var userExpenses []models.PlannedExpense
	for _, expense := range personalExpenses {
		if expense.UserID == userID && expense.GroupID == nil {
			userExpenses = append(userExpenses, expense)
		}
	}

	// Get user's groups
	groups, err := s.groupRepo.FindUserGroups(userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user groups")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get overdue expenses"}
	}

	// Get overdue expenses for each group
	for _, group := range groups {
		groupExpenses, err := s.expenseRepo.FindOverdue(0)
		if err != nil {
			continue
		}

		for _, expense := range groupExpenses {
			if expense.GroupID != nil && *expense.GroupID == group.ID {
				userExpenses = append(userExpenses, expense)
			}
		}
	}

	var response []dto.PlannedExpenseResponse
	for _, expense := range userExpenses {
		response = append(response, *s.mapExpenseToResponse(&expense))
	}

	return response, nil
}

func (s *plannedExpenseService) mapExpenseToResponse(expense *models.PlannedExpense) *dto.PlannedExpenseResponse {
	response := &dto.PlannedExpenseResponse{
		ID:             expense.ID,
		Item:           expense.Item,
		Description:    expense.Description,
		EstimatedPrice: expense.EstimatedPrice,
		ActualPrice:    expense.ActualPrice,
		Category:       expense.Category,
		Status:         expense.Status,
		Priority:       expense.Priority,
		GroupID:        expense.GroupID,
		UserID:         expense.UserID,
		PaidBy:         expense.PaidBy,
		PaidAt:         expense.PaidAt,
		DueDate:        expense.DueDate,
		CreatedAt:      expense.CreatedAt,
		UpdatedAt:      expense.UpdatedAt,
		User: dto.UserResponse{
			ID:          expense.User.ID,
			PhoneNumber: expense.User.PhoneNumber,
			Email:       expense.User.Email,
			FirstName:   expense.User.FirstName,
			LastName:    expense.User.LastName,
			Balance:     expense.User.Balance,
			IsActive:    expense.User.IsActive,
			CreatedAt:   expense.User.CreatedAt.Format(time.RFC3339),
		},
	}

	// Add group info if available
	if expense.GroupID != nil && expense.Group.ID != uuid.Nil {
		response.Group = &dto.GroupResponse{
			ID:          expense.Group.ID,
			Name:        expense.Group.Name,
			Description: expense.Group.Description,
			Balance:     expense.Group.Balance,
			CreatedBy:   expense.Group.CreatedBy,
			IsActive:    expense.Group.IsActive,
			CreatedAt:   expense.Group.CreatedAt.Format(time.RFC3339),
		}
	}

	// Add payer info if available
	if expense.PaidBy != nil && expense.Payer.ID != uuid.Nil {
		response.Payer = &dto.UserResponse{
			ID:          expense.Payer.ID,
			PhoneNumber: expense.Payer.PhoneNumber,
			Email:       expense.Payer.Email,
			FirstName:   expense.Payer.FirstName,
			LastName:    expense.Payer.LastName,
			Balance:     expense.Payer.Balance,
			IsActive:    expense.Payer.IsActive,
			CreatedAt:   expense.Payer.CreatedAt.Format(time.RFC3339),
		}
	}

	return response
}
