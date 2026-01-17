package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlannedExpense struct {
	BaseModel
	Item           string `gorm:"not null" json:"item"`
	Description    string `json:"description"`
	EstimatedPrice int64  `gorm:"not null" json:"estimated_price"` // in cents
	ActualPrice    *int64 `json:"actual_price"`                    // in cents
	Category       string `json:"category"`
	Status         string `gorm:"not null;default:'planned'" json:"status"` // planned, bought, cancelled
	Priority       string `gorm:"default:'medium'" json:"priority"`         // low, medium, high

	// For group expenses
	GroupID *uuid.UUID `gorm:"index" json:"group_id"`

	// For personal expenses
	UserID uuid.UUID `gorm:"not null;index" json:"user_id"`

	// Payment details
	PaidBy *uuid.UUID `gorm:"index" json:"paid_by"`
	PaidAt *time.Time `json:"paid_at"`

	DueDate *time.Time `json:"due_date"`

	// Relationships
	User        *User        `gorm:"foreignKey:UserID" json:"user"`
	Group       *Group       `gorm:"foreignKey:GroupID" json:"group,omitempty"`
	Payer       *User        `gorm:"foreignKey:PaidBy" json:"payer,omitempty"`
	Transaction *Transaction `gorm:"foreignKey:PlannedExpenseID" json:"transaction,omitempty"`
}

func (pe *PlannedExpense) BeforeCreate(tx *gorm.DB) error {
	if pe.ID == uuid.Nil {
		pe.ID = uuid.New()
	}
	return nil
}
