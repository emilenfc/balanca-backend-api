package services

import (
	"balanca/internal/dto"
	"balanca/internal/models"
	"balanca/internal/repositories"
	"balanca/pkg/errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type ReportService interface {
	GetPersonalMonthlyReport(userID uuid.UUID, year, month int) (*dto.MonthlyReportResponse, error)
	GetPersonalDateRangeReport(userID uuid.UUID, startDate, endDate time.Time) (*dto.MonthlyReportResponse, error)
	GetGroupMonthlyReport(userID, groupID uuid.UUID, year, month int) (*dto.GroupReportResponse, error)
	GetGroupDateRangeReport(userID, groupID uuid.UUID, startDate, endDate time.Time) (*dto.GroupReportResponse, error)
	GetCategoryBreakdown(userID uuid.UUID, startDate, endDate time.Time) ([]dto.CategorySummary, error)
	GetSourceBreakdown(userID uuid.UUID, startDate, endDate time.Time) ([]dto.SourceSummary, error)
	GetMemberContributions(groupID uuid.UUID, startDate, endDate time.Time) ([]dto.MemberContribution, error)
}

type reportService struct {
	transactionRepo repositories.TransactionRepository
	userRepo        repositories.UserRepository
	groupRepo       repositories.GroupRepository
}

func NewReportService(
	transactionRepo repositories.TransactionRepository,
	userRepo repositories.UserRepository,
	groupRepo repositories.GroupRepository,
) ReportService {
	return &reportService{
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
		groupRepo:       groupRepo,
	}
}

func (s *reportService) GetPersonalMonthlyReport(userID uuid.UUID, year, month int) (*dto.MonthlyReportResponse, error) {
	// Get date range for the month
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Get transactions for the month
	transactions, err := s.transactionRepo.FindByDateRange("USER", userID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get transactions for report")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Get balance before the month
	balanceBefore, err := s.getBalanceBefore("USER", userID, startDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get starting balance")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Calculate totals
	var totalIncome, totalExpenses int64
	for _, transaction := range transactions {
		if transaction.Type == "CREDIT" {
			totalIncome += transaction.Amount
		} else {
			totalExpenses += transaction.Amount
		}
	}

	endingBalance := balanceBefore + totalIncome - totalExpenses

	// Get category breakdown
	categories, err := s.transactionRepo.GetCategorySummary("USER", userID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get category breakdown")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Get source breakdown
	sources, err := s.transactionRepo.GetSourceSummary("USER", userID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get source breakdown")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Map transactions to response
	var transactionResponses []dto.TransactionResponse
	for _, transaction := range transactions {
		transactionResponses = append(transactionResponses, dto.TransactionResponse{
			ID:          transaction.ID,
			OwnerType:   transaction.OwnerType,
			OwnerID:     transaction.OwnerID,
			Type:        transaction.Type,
			Amount:      transaction.Amount,
			Balance:     transaction.Balance,
			Category:    transaction.Category,
			Source:      transaction.Source,
			Description: transaction.Description,
			CreatedAt:   transaction.CreatedAt.Format(time.RFC3339),
		})
	}

	// Map categories to response
	var categoryResponses []dto.CategorySummary
	totalExpensesFloat := float64(totalExpenses)
	for category, amount := range categories {
		percentage := 0.0
		if totalExpenses > 0 {
			percentage = float64(amount) / totalExpensesFloat * 100
		}
		categoryResponses = append(categoryResponses, dto.CategorySummary{
			Category:   category,
			Amount:     amount,
			Count:      0, // Would need to count separately
			Percentage: percentage,
		})
	}

	// Sort categories by amount (descending)
	sort.Slice(categoryResponses, func(i, j int) bool {
		return categoryResponses[i].Amount > categoryResponses[j].Amount
	})

	// Map sources to response
	var sourceResponses []dto.SourceSummary
	totalIncomeFloat := float64(totalIncome)
	for source, amount := range sources {
		percentage := 0.0
		if totalIncome > 0 {
			percentage = float64(amount) / totalIncomeFloat * 100
		}
		sourceResponses = append(sourceResponses, dto.SourceSummary{
			Source:     source,
			Amount:     amount,
			Count:      0, // Would need to count separately
			Percentage: percentage,
		})
	}

	// Sort sources by amount (descending)
	sort.Slice(sourceResponses, func(i, j int) bool {
		return sourceResponses[i].Amount > sourceResponses[j].Amount
	})

	return &dto.MonthlyReportResponse{
		Month:           startDate.Month().String(),
		Year:            year,
		TotalIncome:     totalIncome,
		TotalExpenses:   totalExpenses,
		NetBalance:      totalIncome - totalExpenses,
		StartingBalance: balanceBefore,
		EndingBalance:   endingBalance,
		Transactions:    transactionResponses,
		Categories:      categoryResponses,
		Sources:         sourceResponses,
	}, nil
}

func (s *reportService) GetPersonalDateRangeReport(userID uuid.UUID, startDate, endDate time.Time) (*dto.MonthlyReportResponse, error) {
	// Similar logic to monthly report but with custom date range
	transactions, err := s.transactionRepo.FindByDateRange("USER", userID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get transactions for report")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Get balance before the period
	balanceBefore, err := s.getBalanceBefore("USER", userID, startDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get starting balance")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Calculate totals
	var totalIncome, totalExpenses int64
	for _, transaction := range transactions {
		if transaction.Type == "CREDIT" {
			totalIncome += transaction.Amount
		} else {
			totalExpenses += transaction.Amount
		}
	}

	endingBalance := balanceBefore + totalIncome - totalExpenses

	// Get category breakdown
	categories, err := s.transactionRepo.GetCategorySummary("USER", userID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get category breakdown")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Get source breakdown
	sources, err := s.transactionRepo.GetSourceSummary("USER", userID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get source breakdown")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Map transactions to response
	var transactionResponses []dto.TransactionResponse
	for _, transaction := range transactions {
		transactionResponses = append(transactionResponses, dto.TransactionResponse{
			ID:          transaction.ID,
			OwnerType:   transaction.OwnerType,
			OwnerID:     transaction.OwnerID,
			Type:        transaction.Type,
			Amount:      transaction.Amount,
			Balance:     transaction.Balance,
			Category:    transaction.Category,
			Source:      transaction.Source,
			Description: transaction.Description,
			CreatedAt:   transaction.CreatedAt.Format(time.RFC3339),
		})
	}

	// Map categories to response
	var categoryResponses []dto.CategorySummary
	totalExpensesFloat := float64(totalExpenses)
	for category, amount := range categories {
		percentage := 0.0
		if totalExpenses > 0 {
			percentage = float64(amount) / totalExpensesFloat * 100
		}
		categoryResponses = append(categoryResponses, dto.CategorySummary{
			Category:   category,
			Amount:     amount,
			Count:      0,
			Percentage: percentage,
		})
	}

	// Sort categories by amount (descending)
	sort.Slice(categoryResponses, func(i, j int) bool {
		return categoryResponses[i].Amount > categoryResponses[j].Amount
	})

	// Map sources to response
	var sourceResponses []dto.SourceSummary
	totalIncomeFloat := float64(totalIncome)
	for source, amount := range sources {
		percentage := 0.0
		if totalIncome > 0 {
			percentage = float64(amount) / totalIncomeFloat * 100
		}
		sourceResponses = append(sourceResponses, dto.SourceSummary{
			Source:     source,
			Amount:     amount,
			Count:      0,
			Percentage: percentage,
		})
	}

	// Sort sources by amount (descending)
	sort.Slice(sourceResponses, func(i, j int) bool {
		return sourceResponses[i].Amount > sourceResponses[j].Amount
	})

	return &dto.MonthlyReportResponse{
		Month:           fmt.Sprintf("%s to %s", startDate.Format("Jan 02"), endDate.Format("Jan 02, 2006")),
		Year:            startDate.Year(),
		TotalIncome:     totalIncome,
		TotalExpenses:   totalExpenses,
		NetBalance:      totalIncome - totalExpenses,
		StartingBalance: balanceBefore,
		EndingBalance:   endingBalance,
		Transactions:    transactionResponses,
		Categories:      categoryResponses,
		Sources:         sourceResponses,
	}, nil
}

func (s *reportService) GetGroupMonthlyReport(userID, groupID uuid.UUID, year, month int) (*dto.GroupReportResponse, error) {
	// Check if user is a member of the group
	userGroup, err := s.groupRepo.FindByUserAndGroup(userID, groupID)
	if err != nil || userGroup.Status != "active" {
		return nil, &errors.AppError{Code: "FORBIDDEN", Message: "You are not a member of this group"}
	}

	// Get group info
	group, err := s.groupRepo.FindByID(groupID)
	if err != nil {
		return nil, &errors.AppError{Code: "GROUP_NOT_FOUND", Message: "Group not found"}
	}

	// Get date range for the month
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Get transactions for the month
	transactions, err := s.transactionRepo.FindByDateRange("GROUP", groupID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get transactions for report")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Get balance before the month
	balanceBefore, err := s.getBalanceBefore("GROUP", groupID, startDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get starting balance")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Calculate totals
	var totalIncome, totalExpenses int64
	for _, transaction := range transactions {
		if transaction.Type == "CREDIT" {
			totalIncome += transaction.Amount
		} else {
			totalExpenses += transaction.Amount
		}
	}

	endingBalance := balanceBefore + totalIncome - totalExpenses

	// Get member contributions
	members, err := s.getMemberContributions(groupID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get member contributions")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Get external sources
	externalSources, err := s.getExternalContributions(groupID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get external contributions")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Get expenses breakdown
	expenses, err := s.getGroupExpensesBreakdown(groupID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get expenses breakdown")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	return &dto.GroupReportResponse{
		GroupID:         groupID,
		GroupName:       group.Name,
		Period:          fmt.Sprintf("%s %d", startDate.Month().String(), year),
		TotalIncome:     totalIncome,
		TotalExpenses:   totalExpenses,
		NetBalance:      totalIncome - totalExpenses,
		StartingBalance: balanceBefore,
		EndingBalance:   endingBalance,
		Members:         members,
		ExternalSources: externalSources,
		Expenses:        expenses,
	}, nil
}

func (s *reportService) GetGroupDateRangeReport(userID, groupID uuid.UUID, startDate, endDate time.Time) (*dto.GroupReportResponse, error) {
	// Check if user is a member of the group
	userGroup, err := s.groupRepo.FindByUserAndGroup(userID, groupID)
	if err != nil || userGroup.Status != "active" {
		return nil, &errors.AppError{Code: "FORBIDDEN", Message: "You are not a member of this group"}
	}

	// Get group info
	group, err := s.groupRepo.FindByID(groupID)
	if err != nil {
		return nil, &errors.AppError{Code: "GROUP_NOT_FOUND", Message: "Group not found"}
	}

	// Get transactions for the period
	transactions, err := s.transactionRepo.FindByDateRange("GROUP", groupID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get transactions for report")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Get balance before the period
	balanceBefore, err := s.getBalanceBefore("GROUP", groupID, startDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get starting balance")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Calculate totals
	var totalIncome, totalExpenses int64
	for _, transaction := range transactions {
		if transaction.Type == "CREDIT" {
			totalIncome += transaction.Amount
		} else {
			totalExpenses += transaction.Amount
		}
	}

	endingBalance := balanceBefore + totalIncome - totalExpenses

	// Get member contributions
	members, err := s.getMemberContributions(groupID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get member contributions")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Get external sources
	externalSources, err := s.getExternalContributions(groupID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get external contributions")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	// Get expenses breakdown
	expenses, err := s.getGroupExpensesBreakdown(groupID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get expenses breakdown")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to generate report"}
	}

	return &dto.GroupReportResponse{
		GroupID:         groupID,
		GroupName:       group.Name,
		Period:          fmt.Sprintf("%s to %s", startDate.Format("Jan 02"), endDate.Format("Jan 02, 2006")),
		TotalIncome:     totalIncome,
		TotalExpenses:   totalExpenses,
		NetBalance:      totalIncome - totalExpenses,
		StartingBalance: balanceBefore,
		EndingBalance:   endingBalance,
		Members:         members,
		ExternalSources: externalSources,
		Expenses:        expenses,
	}, nil
}

func (s *reportService) GetCategoryBreakdown(userID uuid.UUID, startDate, endDate time.Time) ([]dto.CategorySummary, error) {
	categories, err := s.transactionRepo.GetCategorySummary("USER", userID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get category breakdown")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get category breakdown"}
	}

	var response []dto.CategorySummary
	var total int64
	for _, amount := range categories {
		total += amount
	}

	totalFloat := float64(total)
	for category, amount := range categories {
		percentage := 0.0
		if total > 0 {
			percentage = float64(amount) / totalFloat * 100
		}
		response = append(response, dto.CategorySummary{
			Category:   category,
			Amount:     amount,
			Count:      0,
			Percentage: percentage,
		})
	}

	// Sort by amount (descending)
	sort.Slice(response, func(i, j int) bool {
		return response[i].Amount > response[j].Amount
	})

	return response, nil
}

func (s *reportService) GetSourceBreakdown(userID uuid.UUID, startDate, endDate time.Time) ([]dto.SourceSummary, error) {
	sources, err := s.transactionRepo.GetSourceSummary("USER", userID, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get source breakdown")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get source breakdown"}
	}

	var response []dto.SourceSummary
	var total int64
	for _, amount := range sources {
		total += amount
	}

	totalFloat := float64(total)
	for source, amount := range sources {
		percentage := 0.0
		if total > 0 {
			percentage = float64(amount) / totalFloat * 100
		}
		response = append(response, dto.SourceSummary{
			Source:     source,
			Amount:     amount,
			Count:      0,
			Percentage: percentage,
		})
	}

	// Sort by amount (descending)
	sort.Slice(response, func(i, j int) bool {
		return response[i].Amount > response[j].Amount
	})

	return response, nil
}

func (s *reportService) GetMemberContributions(groupID uuid.UUID, startDate, endDate time.Time) ([]dto.MemberContribution, error) {
	return s.getMemberContributions(groupID, startDate, endDate)
}

// Helper methods
func (s *reportService) getBalanceBefore(ownerType string, ownerID uuid.UUID, date time.Time) (int64, error) {
	// Get all transactions before the date
	// In a production system, you might want to cache this or use a more efficient query
	var balance struct {
		Total int64
	}

	// This is a simplified query - in production, you might want to store running balances
	err := s.transactionRepo.GetDB().Model(&models.Transaction{}).
		Select("SUM(CASE WHEN type = 'CREDIT' THEN amount ELSE -amount END) as total").
		Where("owner_type = ? AND owner_id = ? AND created_at < ?", ownerType, ownerID, date).
		Scan(&balance).Error

	return balance.Total, err
}

func (s *reportService) getMemberContributions(groupID uuid.UUID, startDate, endDate time.Time) ([]dto.MemberContribution, error) {
	var contributions []struct {
		PaidBy    uuid.UUID
		FirstName string
		LastName  string
		Total     int64
	}

	err := s.transactionRepo.GetDB().Model(&models.Transaction{}).
		Select("transactions.paid_by, users.first_name, users.last_name, SUM(transactions.amount) as total").
		Joins("LEFT JOIN users ON users.id = transactions.paid_by").
		Where("transactions.owner_type = 'GROUP' AND transactions.owner_id = ? AND transactions.type = 'CREDIT' AND transactions.source = 'member' AND transactions.created_at BETWEEN ? AND ?",
			groupID, startDate, endDate).
		Group("transactions.paid_by, users.first_name, users.last_name").
		Scan(&contributions).Error

	if err != nil {
		return nil, err
	}

	var response []dto.MemberContribution

	// Calculate total contributions
	var total int64
	for _, contribution := range contributions {
		total += contribution.Total
	}

	totalFloat := float64(total)
	for _, contribution := range contributions {
		percentage := 0.0
		if total > 0 {
			percentage = float64(contribution.Total) / totalFloat * 100
		}
		response = append(response, dto.MemberContribution{
			UserID:     contribution.PaidBy,
			FirstName:  contribution.FirstName,
			LastName:   contribution.LastName,
			Amount:     contribution.Total,
			Percentage: percentage,
		})
	}

	// Sort by amount (descending)
	sort.Slice(response, func(i, j int) bool {
		return response[i].Amount > response[j].Amount
	})

	return response, nil
}

func (s *reportService) getExternalContributions(groupID uuid.UUID, startDate, endDate time.Time) ([]dto.ExternalContribution, error) {
	var contributions []struct {
		Source string
		Total  int64
	}

	err := s.transactionRepo.GetDB().Model(&models.Transaction{}).
		Select("source, SUM(amount) as total").
		Where("owner_type = 'GROUP' AND owner_id = ? AND type = 'CREDIT' AND source != 'member' AND created_at BETWEEN ? AND ?",
			groupID, startDate, endDate).
		Group("source").
		Scan(&contributions).Error

	if err != nil {
		return nil, err
	}

	var response []dto.ExternalContribution

	// Calculate total contributions
	var total int64
	for _, contribution := range contributions {
		total += contribution.Total
	}

	totalFloat := float64(total)
	for _, contribution := range contributions {
		percentage := 0.0
		if total > 0 {
			percentage = float64(contribution.Total) / totalFloat * 100
		}
		response = append(response, dto.ExternalContribution{
			Source:     contribution.Source,
			Amount:     contribution.Total,
			Percentage: percentage,
		})
	}

	// Sort by amount (descending)
	sort.Slice(response, func(i, j int) bool {
		return response[i].Amount > response[j].Amount
	})

	return response, nil
}

func (s *reportService) getGroupExpensesBreakdown(groupID uuid.UUID, startDate, endDate time.Time) ([]dto.GroupExpenseSummary, error) {
	var expenses []struct {
		Category  string
		PaidBy    uuid.UUID
		FirstName string
		LastName  string
		Total     int64
		Count     int
	}

	err := s.transactionRepo.GetDB().Model(&models.Transaction{}).
		Select("transactions.category, transactions.paid_by, users.first_name, users.last_name, SUM(transactions.amount) as total, COUNT(*) as count").
		Joins("LEFT JOIN users ON users.id = transactions.paid_by").
		Where("transactions.owner_type = 'GROUP' AND transactions.owner_id = ? AND transactions.type = 'DEBIT' AND transactions.created_at BETWEEN ? AND ?",
			groupID, startDate, endDate).
		Group("transactions.category, transactions.paid_by, users.first_name, users.last_name").
		Scan(&expenses).Error

	if err != nil {
		return nil, err
	}

	var response []dto.GroupExpenseSummary
	for _, expense := range expenses {
		response = append(response, dto.GroupExpenseSummary{
			Category:  expense.Category,
			Amount:    expense.Total,
			Count:     expense.Count,
			PaidBy:    expense.PaidBy,
			PayerName: expense.FirstName + " " + expense.LastName,
		})
	}

	// Sort by amount (descending)
	sort.Slice(response, func(i, j int) bool {
		return response[i].Amount > response[j].Amount
	})

	return response, nil
}
