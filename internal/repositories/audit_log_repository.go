package repositories

import (
	"balanca/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuditLogRepository interface {
	Create(log *models.AuditLog) error
	FindByEntity(entity string, entityID uuid.UUID, page, limit int) ([]models.AuditLog, int64, error)
	FindByGroup(groupID uuid.UUID, page, limit int) ([]models.AuditLog, int64, error)
	FindByUser(userID uuid.UUID, page, limit int) ([]models.AuditLog, int64, error)
	FindByDateRange(startDate, endDate time.Time, page, limit int) ([]models.AuditLog, int64, error)
}

type auditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(log *models.AuditLog) error {
	return r.db.Create(log).Error
}

func (r *auditLogRepository) FindByEntity(entity string, entityID uuid.UUID, page, limit int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	offset := (page - 1) * limit
	query := r.db.Preload("User").Preload("Group").
		Where("entity = ? AND entity_id = ?", entity, entityID).
		Order("performed_at DESC")

	err := query.Model(&models.AuditLog{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}

func (r *auditLogRepository) FindByGroup(groupID uuid.UUID, page, limit int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	offset := (page - 1) * limit
	query := r.db.Preload("User").Preload("Group").
		Where("group_id = ?", groupID).
		Order("performed_at DESC")

	err := query.Model(&models.AuditLog{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}

func (r *auditLogRepository) FindByUser(userID uuid.UUID, page, limit int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	offset := (page - 1) * limit
	query := r.db.Preload("User").Preload("Group").
		Where("performed_by = ?", userID).
		Order("performed_at DESC")

	err := query.Model(&models.AuditLog{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}

func (r *auditLogRepository) FindByDateRange(startDate, endDate time.Time, page, limit int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	offset := (page - 1) * limit
	query := r.db.Preload("User").Preload("Group").
		Where("performed_at BETWEEN ? AND ?", startDate, endDate).
		Order("performed_at DESC")

	err := query.Model(&models.AuditLog{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}