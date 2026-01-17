package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Notification struct {
	BaseModel
	UserID  uuid.UUID              `gorm:"not null;index" json:"user_id"`
	Type    string                 `gorm:"not null" json:"type"` // group_invite, transaction, expense_paid, etc.
	Title   string                 `gorm:"not null" json:"title"`
	Message string                 `gorm:"not null" json:"message"`
	Data    map[string]interface{} `gorm:"type:jsonb" json:"data"`
	IsRead  bool                   `gorm:"default:false" json:"is_read"`
	ReadAt  *time.Time             `json:"read_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}
