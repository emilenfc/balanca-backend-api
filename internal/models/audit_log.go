package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuditLog struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Entity       string         `gorm:"not null;index" json:"entity"` // user, group, transaction, etc.
	EntityID     uuid.UUID      `gorm:"not null;index" json:"entity_id"`
	Action       string         `gorm:"not null" json:"action"` // create, update, delete
	Changes      map[string]interface{} `gorm:"type:jsonb" json:"changes"`
	PerformedBy  uuid.UUID      `gorm:"not null;index" json:"performed_by"`
	PerformedAt  time.Time      `json:"performed_at"`
	
	// For group actions
	GroupID      *uuid.UUID    `gorm:"index" json:"group_id"`
	
	// Relationships
	User         User          `gorm:"foreignKey:PerformedBy" json:"user"`
	Group        *Group         `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}

func (al *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if al.ID == uuid.Nil {
		al.ID = uuid.New()
	}
	if al.PerformedAt.IsZero() {
		al.PerformedAt = time.Now()
	}
	return nil
}