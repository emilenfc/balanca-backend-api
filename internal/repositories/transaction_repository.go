package repositories

import (
	"balanca/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TransactionRepository interface {
	Create(transaction *models.Transaction) error
	FindByID(id uuid.UUID) (*models.Transaction, error)
	FindByOwner(ownerType string, ownerID uuid.UUID, page, limit int) ([]models.Transaction, int64, error)
	FindByUser(userID uuid.UUID, page, limit int) ([]models.Transaction, int64, error)
	FindByGroup(groupID uuid.UUID, page, limit int) ([]models.Transaction, int64, error)
	FindByDateRange(ownerType string, ownerID uuid.UUID, startDate, endDate time.Time) ([]models.Transaction, error)
	GetBalance(ownerType string, ownerID uuid.UUID) (int64, error)
	GetMonthlySummary(ownerType string, ownerID uuid.UUID, year int, month int) (*models.Transaction, error)
	GetCategorySummary(ownerType string, ownerID uuid.UUID, startDate, endDate time.Time) (map[string]int64, error)
	GetSourceSummary(ownerType string, ownerID uuid.UUID, startDate, endDate time.Time) (map[string]int64, error)
	GetDB() *gorm.DB
}

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) Create(transaction *models.Transaction) error {
	return r.db.Create(transaction).Error
}
// Add this method
func (r *transactionRepository) GetDB() *gorm.DB {
	return r.db
}

func (r *transactionRepository) FindByID(id uuid.UUID) (*models.Transaction, error) {
	var transaction models.Transaction
	err := r.db.Preload("User").Preload("Group").Preload("Payer").
		Where("id = ?", id).First(&transaction).Error
	return &transaction, err
}

func (r *transactionRepository) FindByOwner(ownerType string, ownerID uuid.UUID, page, limit int) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	var total int64

	offset := (page - 1) * limit
	query := r.db.Preload("User").Preload("Group").Preload("Payer").
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Order("created_at DESC")

	err := query.Model(&models.Transaction{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(offset).Limit(limit).Find(&transactions).Error
	return transactions, total, err
}

func (r *transactionRepository) FindByUser(userID uuid.UUID, page, limit int) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	var total int64

	offset := (page - 1) * limit
	query := r.db.Preload("User").Preload("Group").Preload("Payer").
		Where("user_id = ?", userID).
		Order("created_at DESC")

	err := query.Model(&models.Transaction{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(offset).Limit(limit).Find(&transactions).Error
	return transactions, total, err
}

func (r *transactionRepository) FindByGroup(groupID uuid.UUID, page, limit int) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	var total int64

	offset := (page - 1) * limit
	query := r.db.Preload("User").Preload("Group").Preload("Payer").
		Where("group_id = ?", groupID).
		Order("created_at DESC")

	err := query.Model(&models.Transaction{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(offset).Limit(limit).Find(&transactions).Error
	return transactions, total, err
}

func (r *transactionRepository) FindByDateRange(ownerType string, ownerID uuid.UUID, startDate, endDate time.Time) ([]models.Transaction, error) {
	var transactions []models.Transaction
	err := r.db.Preload("User").Preload("Group").Preload("Payer").
		Where("owner_type = ? AND owner_id = ? AND created_at BETWEEN ? AND ?", 
			ownerType, ownerID, startDate, endDate).
		Order("created_at ASC").
		Find(&transactions).Error
	return transactions, err
}

func (r *transactionRepository) GetBalance(ownerType string, ownerID uuid.UUID) (int64, error) {
	var balance struct {
		Total int64
	}
	
	err := r.db.Model(&models.Transaction{}).
		Select("SUM(CASE WHEN type = 'CREDIT' THEN amount ELSE -amount END) as total").
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Scan(&balance).Error
	
	return balance.Total, err
}

func (r *transactionRepository) GetMonthlySummary(ownerType string, ownerID uuid.UUID, year int, month int) (*models.Transaction, error) {
	var summary struct {
		TotalIncome   int64
		TotalExpenses int64
	}
	
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)
	
	err := r.db.Model(&models.Transaction{}).
		Select(
			"SUM(CASE WHEN type = 'CREDIT' THEN amount ELSE 0 END) as total_income",
			"SUM(CASE WHEN type = 'DEBIT' THEN amount ELSE 0 END) as total_expenses",
		).
		Where("owner_type = ? AND owner_id = ? AND created_at BETWEEN ? AND ?", 
			ownerType, ownerID, startDate, endDate).
		Scan(&summary).Error
	
	// Create a transaction model for the summary
	transaction := &models.Transaction{
		OwnerType: ownerType,
		OwnerID:   ownerID,
		Amount:    summary.TotalIncome - summary.TotalExpenses,
		Metadata: map[string]interface{}{
			"total_income":   summary.TotalIncome,
			"total_expenses": summary.TotalExpenses,
		},
	}
	
	return transaction, err
}

func (r *transactionRepository) GetCategorySummary(ownerType string, ownerID uuid.UUID, startDate, endDate time.Time) (map[string]int64, error) {
	var results []struct {
		Category string
		Total    int64
	}
	
	summary := make(map[string]int64)
	
	err := r.db.Model(&models.Transaction{}).
		Select("category, SUM(amount) as total").
		Where("owner_type = ? AND owner_id = ? AND type = 'DEBIT' AND created_at BETWEEN ? AND ?", 
			ownerType, ownerID, startDate, endDate).
		Group("category").
		Find(&results).Error
	
	for _, result := range results {
		summary[result.Category] = result.Total
	}
	
	return summary, err
}

func (r *transactionRepository) GetSourceSummary(ownerType string, ownerID uuid.UUID, startDate, endDate time.Time) (map[string]int64, error) {
	var results []struct {
		Source string
		Total  int64
	}
	
	summary := make(map[string]int64)
	
	err := r.db.Model(&models.Transaction{}).
		Select("source, SUM(amount) as total").
		Where("owner_type = ? AND owner_id = ? AND type = 'CREDIT' AND created_at BETWEEN ? AND ?", 
			ownerType, ownerID, startDate, endDate).
		Group("source").
		Find(&results).Error
	
	for _, result := range results {
		summary[result.Source] = result.Total
	}
	
	return summary, err
}