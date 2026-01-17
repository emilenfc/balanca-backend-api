package repositories

import (
	"balanca/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlannedExpenseRepository interface {
	Create(expense *models.PlannedExpense) error
	FindByID(id uuid.UUID) (*models.PlannedExpense, error)
	FindByUser(userID uuid.UUID, status string, page, limit int) ([]models.PlannedExpense, int64, error)
	FindByGroup(groupID uuid.UUID, status string, page, limit int) ([]models.PlannedExpense, int64, error)
	Update(expense *models.PlannedExpense) error
	Delete(id uuid.UUID) error
	MarkAsBought(id uuid.UUID, actualPrice int64, paidBy uuid.UUID) error
	MarkAsCancelled(id uuid.UUID) error
	FindOverdue(days int) ([]models.PlannedExpense, error)
}

type plannedExpenseRepository struct {
	db *gorm.DB
}

func NewPlannedExpenseRepository(db *gorm.DB) PlannedExpenseRepository {
	return &plannedExpenseRepository{db: db}
}

func (r *plannedExpenseRepository) Create(expense *models.PlannedExpense) error {
	return r.db.Create(expense).Error
}

func (r *plannedExpenseRepository) FindByID(id uuid.UUID) (*models.PlannedExpense, error) {
	var expense models.PlannedExpense
	err := r.db.Preload("User").Preload("Group").Preload("Payer").
		Where("id = ?", id).First(&expense).Error
	return &expense, err
}

func (r *plannedExpenseRepository) FindByUser(userID uuid.UUID, status string, page, limit int) ([]models.PlannedExpense, int64, error) {
	var expenses []models.PlannedExpense
	var total int64

	offset := (page - 1) * limit
	query := r.db.Preload("User").Preload("Group").Preload("Payer").
		Where("user_id = ?", userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query = query.Order("created_at DESC")

	err := query.Model(&models.PlannedExpense{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(offset).Limit(limit).Find(&expenses).Error
	return expenses, total, err
}

func (r *plannedExpenseRepository) FindByGroup(groupID uuid.UUID, status string, page, limit int) ([]models.PlannedExpense, int64, error) {
	var expenses []models.PlannedExpense
	var total int64

	offset := (page - 1) * limit
	query := r.db.Preload("User").Preload("Group").Preload("Payer").
		Where("group_id = ?", groupID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query = query.Order("created_at DESC")

	err := query.Model(&models.PlannedExpense{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(offset).Limit(limit).Find(&expenses).Error
	return expenses, total, err
}

func (r *plannedExpenseRepository) Update(expense *models.PlannedExpense) error {
	return r.db.Save(expense).Error
}

func (r *plannedExpenseRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.PlannedExpense{}, "id = ?", id).Error
}

func (r *plannedExpenseRepository) MarkAsBought(id uuid.UUID, actualPrice int64, paidBy uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&models.PlannedExpense{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       "bought",
			"actual_price": actualPrice,
			"paid_by":      paidBy,
			"paid_at":      now,
			"updated_at":   now,
		}).Error
}

func (r *plannedExpenseRepository) MarkAsCancelled(id uuid.UUID) error {
	return r.db.Model(&models.PlannedExpense{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     "cancelled",
			"updated_at": time.Now(),
		}).Error
}

func (r *plannedExpenseRepository) FindOverdue(days int) ([]models.PlannedExpense, error) {
	var expenses []models.PlannedExpense
	cutoffDate := time.Now().AddDate(0, 0, -days)
	
	err := r.db.Preload("User").Preload("Group").
		Where("status = 'planned' AND due_date < ?", cutoffDate).
		Find(&expenses).Error
	
	return expenses, err
}