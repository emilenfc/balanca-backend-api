package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreatePlannedExpenseRequest struct {
	Item          string     `json:"item" binding:"required"`
	Description   string     `json:"description"`
	EstimatedPrice int64     `json:"estimated_price" binding:"required,gt=0"`
	Category      string     `json:"category" binding:"required"`
	Priority      string     `json:"priority" binding:"oneof=low medium high"`
	
	// For group expenses
	GroupID       *uuid.UUID `json:"group_id"`
	
	DueDate       *time.Time `json:"due_date"`
}

type PlannedExpenseResponse struct {
	ID            uuid.UUID  `json:"id"`
	Item          string     `json:"item"`
	Description   string     `json:"description"`
	EstimatedPrice int64     `json:"estimated_price"`
	ActualPrice   *int64     `json:"actual_price"`
	Category      string     `json:"category"`
	Status        string     `json:"status"`
	Priority      string     `json:"priority"`
	
	GroupID       *uuid.UUID `json:"group_id,omitempty"`
	UserID        uuid.UUID  `json:"user_id"`
	
	PaidBy        *uuid.UUID `json:"paid_by,omitempty"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	
	DueDate       *time.Time `json:"due_date,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	
	User          UserResponse       `json:"user"`
	Group         *GroupResponse     `json:"group,omitempty"`
	Payer         *UserResponse      `json:"payer,omitempty"`
	Transaction   *TransactionResponse `json:"transaction,omitempty"`
}

type UpdatePlannedExpenseRequest struct {
	Item          *string    `json:"item"`
	Description   *string    `json:"description"`
	EstimatedPrice *int64    `json:"estimated_price"`
	Category      *string    `json:"category"`
	Priority      *string    `json:"priority"`
	DueDate       *time.Time `json:"due_date"`
}

type MarkAsBoughtRequest struct {
	ActualPrice int64 `json:"actual_price" binding:"required,gt=0"`
}