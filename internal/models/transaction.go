package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Transaction struct {
	BaseModel
	OwnerType   string                 `gorm:"not null;index" json:"owner_type"` // USER, GROUP
	OwnerID     uuid.UUID              `gorm:"not null;index" json:"owner_id"`
	Type        string                 `gorm:"not null" json:"type"`    // CREDIT, DEBIT
	Amount      int64                  `gorm:"not null" json:"amount"`  // in cents
	Balance     int64                  `gorm:"not null" json:"balance"` // balance after transaction
	Category    string                 `json:"category"`                // food, transport, home, personal, etc.
	Source      string                 `json:"source"`                  // salary, gig, gift, transfer, etc.
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `gorm:"type:jsonb" json:"metadata"`

	// For group transactions
	GroupID          *uuid.UUID `gorm:"index" json:"group_id"`
	PaidBy           *uuid.UUID `gorm:"index" json:"paid_by"`
	PlannedExpenseID *uuid.UUID `gorm:"index" json:"planned_expense_id"`

	// For personal transactions
	UserID uuid.UUID `gorm:"not null;index" json:"user_id"`

	// Relationships
	User           User           `gorm:"foreignKey:UserID" json:"user"`
	Group          Group          `gorm:"foreignKey:GroupID" json:"group,omitempty"`
	Payer          User           `gorm:"foreignKey:PaidBy" json:"payer,omitempty"`
	PlannedExpense PlannedExpense `gorm:"foreignKey:PlannedExpenseID" json:"planned_expense,omitempty"`
}

func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
