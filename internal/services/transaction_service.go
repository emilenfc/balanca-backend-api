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

type TransactionService interface {
	CreatePersonalTransaction(userID uuid.UUID, req dto.CreateTransactionRequest) (*dto.TransactionResponse, error)
	CreateGroupTransaction(userID uuid.UUID, req dto.CreateTransactionRequest) (*dto.TransactionResponse, error)
	GetPersonalTransactions(userID uuid.UUID, page, limit int) ([]dto.TransactionResponse, int64, error)
	GetGroupTransactions(userID, groupID uuid.UUID, page, limit int) ([]dto.TransactionResponse, int64, error)
	GetTransaction(userID, transactionID uuid.UUID) (*dto.TransactionResponse, error)
	TransferToGroup(userID uuid.UUID, req dto.TransferToGroupRequest) (*dto.TransactionResponse, error)
	PayGroupExpense(userID, groupID uuid.UUID, req dto.PayGroupExpenseRequest) (*dto.TransactionResponse, error)
	RecordExternalIncome(userID, groupID uuid.UUID, amount int64, source string) (*dto.TransactionResponse, error)
}

type transactionService struct {
	transactionRepo repositories.TransactionRepository
	userRepo        repositories.UserRepository
	groupRepo       repositories.GroupRepository
	expenseRepo     repositories.PlannedExpenseRepository
	auditRepo       repositories.AuditLogRepository
	db              *gorm.DB
}

func NewTransactionService(
	transactionRepo repositories.TransactionRepository,
	userRepo repositories.UserRepository,
	groupRepo repositories.GroupRepository,
	expenseRepo repositories.PlannedExpenseRepository,
	auditRepo repositories.AuditLogRepository,
	db *gorm.DB,
) TransactionService {
	return &transactionService{
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
		groupRepo:       groupRepo,
		expenseRepo:     expenseRepo,
		auditRepo:       auditRepo,
		db:              db,
	}
}

func (s *transactionService) CreatePersonalTransaction(userID uuid.UUID, req dto.CreateTransactionRequest) (*dto.TransactionResponse, error) {
	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get user and current balance
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		tx.Rollback()
		return nil, &errors.AppError{Code: "USER_NOT_FOUND", Message: "User not found"}
	}

	// Calculate new balance
	var newBalance int64
	if req.Type == "CREDIT" {
		newBalance = user.Balance + req.Amount
	} else { // DEBIT
		if user.Balance < req.Amount {
			tx.Rollback()
			return nil, &errors.AppError{Code: "INSUFFICIENT_BALANCE", Message: "Insufficient balance"}
		}
		newBalance = user.Balance - req.Amount
	}

	// Create transaction
	transaction := &models.Transaction{
		OwnerType:   "USER",
		OwnerID:     userID,
		Type:        req.Type,
		Amount:      req.Amount,
		Balance:     newBalance,
		Category:    req.Category,
		Source:      req.Source,
		Description: req.Description,
		UserID:      userID,
		Metadata: map[string]interface{}{
			"personal": true,
		},
	}

	if err := tx.Create(transaction).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to create transaction")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create transaction"}
	}

	// Update user balance
	user.Balance = newBalance
	if err := tx.Save(user).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to update user balance")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create transaction"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "transaction",
		EntityID:    transaction.ID,
		Action:      "create",
		Changes:     map[string]interface{}{"type": req.Type, "amount": req.Amount},
		PerformedBy: userID,
	}

	if err := tx.Create(auditLog).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to create audit log")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create transaction"}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to commit transaction")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create transaction"}
	}

	// Get full transaction data
	fullTransaction, err := s.transactionRepo.FindByID(transaction.ID)
	if err != nil {
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get transaction data"}
	}

	return s.mapTransactionToResponse(fullTransaction), nil
}

func (s *transactionService) CreateGroupTransaction(userID uuid.UUID, req dto.CreateTransactionRequest) (*dto.TransactionResponse, error) {
	if req.GroupID == nil {
		return nil, &errors.AppError{Code: "INVALID_REQUEST", Message: "Group ID is required"}
	}

	// Check if user is a member of the group
	userGroup, err := s.groupRepo.FindByUserAndGroup(userID, *req.GroupID)
	if err != nil || userGroup.Status != "active" {
		return nil, &errors.AppError{Code: "FORBIDDEN", Message: "You are not a member of this group"}
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get group and current balance
	group, err := s.groupRepo.FindByID(*req.GroupID)
	if err != nil {
		tx.Rollback()
		return nil, &errors.AppError{Code: "GROUP_NOT_FOUND", Message: "Group not found"}
	}

	// Calculate new balance
	var newBalance int64
	if req.Type == "CREDIT" {
		newBalance = group.Balance + req.Amount
	} else { // DEBIT
		if group.Balance < req.Amount {
			tx.Rollback()
			return nil, &errors.AppError{Code: "INSUFFICIENT_BALANCE", Message: "Insufficient group balance"}
		}
		newBalance = group.Balance - req.Amount
	}

	// Create transaction
	transaction := &models.Transaction{
		OwnerType:   "GROUP",
		OwnerID:     *req.GroupID,
		Type:        req.Type,
		Amount:      req.Amount,
		Balance:     newBalance,
		Category:    req.Category,
		Source:      req.Source,
		Description: req.Description,
		GroupID:     req.GroupID,
		PaidBy:      req.PaidBy,
		UserID:      userID,
		Metadata: map[string]interface{}{
			"group": true,
		},
	}

	if req.PlannedExpenseID != nil {
		transaction.PlannedExpenseID = req.PlannedExpenseID
	}

	if err := tx.Create(transaction).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to create transaction")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create transaction"}
	}

	// Update group balance
	group.Balance = newBalance
	if err := tx.Save(group).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to update group balance")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create transaction"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "transaction",
		EntityID:    transaction.ID,
		Action:      "create",
		Changes:     map[string]interface{}{"type": req.Type, "amount": req.Amount},
		PerformedBy: userID,
		GroupID:     req.GroupID,
	}

	if err := tx.Create(auditLog).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to create audit log")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create transaction"}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to commit transaction")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create transaction"}
	}

	// Get full transaction data
	fullTransaction, err := s.transactionRepo.FindByID(transaction.ID)
	if err != nil {
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get transaction data"}
	}

	return s.mapTransactionToResponse(fullTransaction), nil
}

func (s *transactionService) TransferToGroup(userID uuid.UUID, req dto.TransferToGroupRequest) (*dto.TransactionResponse, error) {
	// Check if user is a member of the group
	userGroup, err := s.groupRepo.FindByUserAndGroup(userID, req.GroupID)
	if err != nil || userGroup.Status != "active" {
		return nil, &errors.AppError{Code: "FORBIDDEN", Message: "You are not a member of this group"}
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get user and check balance
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		tx.Rollback()
		return nil, &errors.AppError{Code: "USER_NOT_FOUND", Message: "User not found"}
	}

	if user.Balance < req.Amount {
		tx.Rollback()
		return nil, &errors.AppError{Code: "INSUFFICIENT_BALANCE", Message: "Insufficient personal balance"}
	}

	// Get group
	group, err := s.groupRepo.FindByID(req.GroupID)
	if err != nil {
		tx.Rollback()
		return nil, &errors.AppError{Code: "GROUP_NOT_FOUND", Message: "Group not found"}
	}

	// Update user balance (debit)
	user.Balance -= req.Amount

	// Create personal transaction (debit)
	personalTransaction := &models.Transaction{
		OwnerType:   "USER",
		OwnerID:     userID,
		Type:        "DEBIT",
		Amount:      req.Amount,
		Balance:     user.Balance,
		Category:    "transfer",
		Source:      "group_transfer",
		Description: req.Description,
		GroupID:     &req.GroupID,
		UserID:      userID,
		Metadata: map[string]interface{}{
			"transfer_to_group": true,
			"group_id":          req.GroupID.String(),
		},
	}

	if err := tx.Create(personalTransaction).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to create personal transaction")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to transfer money"}
	}

	// Update group balance (credit)
	group.Balance += req.Amount

	// Create group transaction (credit)
	groupTransaction := &models.Transaction{
		OwnerType:   "GROUP",
		OwnerID:     req.GroupID,
		Type:        "CREDIT",
		Amount:      req.Amount,
		Balance:     group.Balance,
		Category:    "member_contribution",
		Source:      "member",
		Description: req.Description,
		GroupID:     &req.GroupID,
		PaidBy:      &userID,
		UserID:      userID,
		Metadata: map[string]interface{}{
			"from_member": true,
			"member_id":   userID.String(),
		},
	}

	if err := tx.Create(groupTransaction).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to create group transaction")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to transfer money"}
	}

	// Save updated balances
	if err := tx.Save(user).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to update user balance")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to transfer money"}
	}

	if err := tx.Save(group).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to update group balance")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to transfer money"}
	}

	// Create audit logs
	personalAuditLog := &models.AuditLog{
		Entity:      "transaction",
		EntityID:    personalTransaction.ID,
		Action:      "transfer_to_group",
		Changes:     map[string]interface{}{"amount": req.Amount, "group_id": req.GroupID.String()},
		PerformedBy: userID,
	}

	groupAuditLog := &models.AuditLog{
		Entity:      "transaction",
		EntityID:    groupTransaction.ID,
		Action:      "receive_from_member",
		Changes:     map[string]interface{}{"amount": req.Amount, "member_id": userID.String()},
		PerformedBy: userID,
		GroupID:     &req.GroupID,
	}

	if err := tx.Create(personalAuditLog).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to create audit log")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to transfer money"}
	}

	if err := tx.Create(groupAuditLog).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to create audit log")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to transfer money"}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to commit transaction")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to transfer money"}
	}

	// Get full transaction data
	fullTransaction, err := s.transactionRepo.FindByID(groupTransaction.ID)
	if err != nil {
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get transaction data"}
	}

	return s.mapTransactionToResponse(fullTransaction), nil
}

func (s *transactionService) PayGroupExpense(userID, groupID uuid.UUID, req dto.PayGroupExpenseRequest) (*dto.TransactionResponse, error) {
	// Check if user is a member of the group
	userGroup, err := s.groupRepo.FindByUserAndGroup(userID, groupID)
	if err != nil || userGroup.Status != "active" {
		return nil, &errors.AppError{Code: "FORBIDDEN", Message: "You are not a member of this group"}
	}

	// Get planned expense
	expense, err := s.expenseRepo.FindByID(req.PlannedExpenseID)
	if err != nil {
		return nil, &errors.AppError{Code: "EXPENSE_NOT_FOUND", Message: "Planned expense not found"}
	}

	if expense.GroupID == nil || *expense.GroupID != groupID {
		return nil, &errors.AppError{Code: "FORBIDDEN", Message: "Expense does not belong to this group"}
	}

	if expense.Status != "planned" {
		return nil, &errors.AppError{Code: "INVALID_STATUS", Message: "Expense is not in planned status"}
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get group and check balance
	group, err := s.groupRepo.FindByID(groupID)
	if err != nil {
		tx.Rollback()
		return nil, &errors.AppError{Code: "GROUP_NOT_FOUND", Message: "Group not found"}
	}

	if group.Balance < req.ActualPrice {
		tx.Rollback()
		return nil, &errors.AppError{Code: "INSUFFICIENT_BALANCE", Message: "Insufficient group balance"}
	}

	// Update group balance
	group.Balance -= req.ActualPrice

	// Create group transaction (debit)
	transaction := &models.Transaction{
		OwnerType:        "GROUP",
		OwnerID:          groupID,
		Type:             "DEBIT",
		Amount:           req.ActualPrice,
		Balance:          group.Balance,
		Category:         expense.Category,
		Source:           "expense_payment",
		Description:      req.Description,
		GroupID:          &groupID,
		PaidBy:           &userID,
		PlannedExpenseID: &req.PlannedExpenseID,
		UserID:           userID,
		Metadata: map[string]interface{}{
			"expense_payment": true,
			"expense_id":      req.PlannedExpenseID.String(),
		},
	}

	if err := tx.Create(transaction).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to create transaction")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to pay expense"}
	}

	// Update planned expense
	now := time.Now()
	expense.Status = "bought"
	expense.ActualPrice = &req.ActualPrice
	expense.PaidBy = &userID
	expense.PaidAt = &now

	if err := tx.Save(expense).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to update planned expense")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to pay expense"}
	}

	// Save updated group balance
	if err := tx.Save(group).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to update group balance")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to pay expense"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "planned_expense",
		EntityID:    expense.ID,
		Action:      "mark_as_paid",
		Changes:     map[string]interface{}{"actual_price": req.ActualPrice, "paid_by": userID.String()},
		PerformedBy: userID,
		GroupID:     &groupID,
	}

	if err := tx.Create(auditLog).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to create audit log")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to pay expense"}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to commit transaction")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to pay expense"}
	}

	// Get full transaction data
	fullTransaction, err := s.transactionRepo.FindByID(transaction.ID)
	if err != nil {
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get transaction data"}
	}

	return s.mapTransactionToResponse(fullTransaction), nil
}

func (s *transactionService) RecordExternalIncome(userID, groupID uuid.UUID, amount int64, source string) (*dto.TransactionResponse, error) {
	// Check if user is a member of the group
	userGroup, err := s.groupRepo.FindByUserAndGroup(userID, groupID)
	if err != nil || userGroup.Status != "active" {
		return nil, &errors.AppError{Code: "FORBIDDEN", Message: "You are not a member of this group"}
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get group
	group, err := s.groupRepo.FindByID(groupID)
	if err != nil {
		tx.Rollback()
		return nil, &errors.AppError{Code: "GROUP_NOT_FOUND", Message: "Group not found"}
	}

	// Update group balance
	group.Balance += amount

	// Create group transaction (credit)
	transaction := &models.Transaction{
		OwnerType:   "GROUP",
		OwnerID:     groupID,
		Type:        "CREDIT",
		Amount:      amount,
		Balance:     group.Balance,
		Category:    "external_income",
		Source:      source,
		Description: "External contribution",
		GroupID:     &groupID,
		UserID:      userID,
		Metadata: map[string]interface{}{
			"external_income": true,
			"source":          source,
		},
	}

	if err := tx.Create(transaction).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to create transaction")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to record income"}
	}

	// Save updated group balance
	if err := tx.Save(group).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to update group balance")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to record income"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "transaction",
		EntityID:    transaction.ID,
		Action:      "record_external_income",
		Changes:     map[string]interface{}{"amount": amount, "source": source},
		PerformedBy: userID,
		GroupID:     &groupID,
	}

	if err := tx.Create(auditLog).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to create audit log")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to record income"}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to commit transaction")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to record income"}
	}

	// Get full transaction data
	fullTransaction, err := s.transactionRepo.FindByID(transaction.ID)
	if err != nil {
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get transaction data"}
	}

	return s.mapTransactionToResponse(fullTransaction), nil
}

func (s *transactionService) GetPersonalTransactions(userID uuid.UUID, page, limit int) ([]dto.TransactionResponse, int64, error) {
	transactions, total, err := s.transactionRepo.FindByUser(userID, page, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get personal transactions")
		return nil, 0, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get transactions"}
	}

	var response []dto.TransactionResponse
	for _, transaction := range transactions {
		response = append(response, *s.mapTransactionToResponse(&transaction))
	}

	return response, total, nil
}

func (s *transactionService) GetGroupTransactions(userID, groupID uuid.UUID, page, limit int) ([]dto.TransactionResponse, int64, error) {
	// Check if user is a member of the group
	userGroup, err := s.groupRepo.FindByUserAndGroup(userID, groupID)
	if err != nil || userGroup.Status != "active" {
		return nil, 0, &errors.AppError{Code: "FORBIDDEN", Message: "You are not a member of this group"}
	}

	transactions, total, err := s.transactionRepo.FindByGroup(groupID, page, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get group transactions")
		return nil, 0, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get transactions"}
	}

	var response []dto.TransactionResponse
	for _, transaction := range transactions {
		response = append(response, *s.mapTransactionToResponse(&transaction))
	}

	return response, total, nil
}

func (s *transactionService) GetTransaction(userID, transactionID uuid.UUID) (*dto.TransactionResponse, error) {
	transaction, err := s.transactionRepo.FindByID(transactionID)
	if err != nil {
		return nil, &errors.AppError{Code: "TRANSACTION_NOT_FOUND", Message: "Transaction not found"}
	}

	// Check if user has access to this transaction
	if transaction.OwnerType == "USER" && transaction.UserID != userID {
		return nil, &errors.AppError{Code: "FORBIDDEN", Message: "Access denied"}
	}

	if transaction.OwnerType == "GROUP" && transaction.GroupID != nil {
		// Check if user is a member of the group
		userGroup, err := s.groupRepo.FindByUserAndGroup(userID, *transaction.GroupID)
		if err != nil || userGroup.Status != "active" {
			return nil, &errors.AppError{Code: "FORBIDDEN", Message: "Access denied"}
		}
	}

	return s.mapTransactionToResponse(transaction), nil
}

func (s *transactionService) mapTransactionToResponse(transaction *models.Transaction) *dto.TransactionResponse {
	response := &dto.TransactionResponse{
		ID:               transaction.ID,
		OwnerType:        transaction.OwnerType,
		OwnerID:          transaction.OwnerID,
		Type:             transaction.Type,
		Amount:           transaction.Amount,
		Balance:          transaction.Balance,
		Category:         transaction.Category,
		Source:           transaction.Source,
		Description:      transaction.Description,
		CreatedAt:        transaction.CreatedAt.Format(time.RFC3339),
		GroupID:          transaction.GroupID,
		PaidBy:           transaction.PaidBy,
		PlannedExpenseID: transaction.PlannedExpenseID,
	}

	// Add user info if available
	if transaction.User.ID != uuid.Nil {
		response.Payer = &dto.UserResponse{
			ID:          transaction.User.ID,
			PhoneNumber: transaction.User.PhoneNumber,
			Email:       transaction.User.Email,
			FirstName:   transaction.User.FirstName,
			LastName:    transaction.User.LastName,
			Balance:     transaction.User.Balance,
			IsActive:    transaction.User.IsActive,
			CreatedAt:   transaction.User.CreatedAt.Format(time.RFC3339),
		}
	}

	// Add group info if available
	if transaction.GroupID != nil && transaction.Group.ID != uuid.Nil {
		response.Group = &dto.GroupResponse{
			ID:          transaction.Group.ID,
			Name:        transaction.Group.Name,
			Description: transaction.Group.Description,
			Balance:     transaction.Group.Balance,
			CreatedBy:   transaction.Group.CreatedBy,
			IsActive:    transaction.Group.IsActive,
			CreatedAt:   transaction.Group.CreatedAt.Format(time.RFC3339),
		}
	}

	return response
}
