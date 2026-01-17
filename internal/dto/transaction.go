package dto

import "github.com/google/uuid"

type CreateTransactionRequest struct {
	Type        string `json:"type" binding:"required,oneof=CREDIT DEBIT"`
	Amount      int64  `json:"amount" binding:"required,gt=0"`
	Category    string `json:"category" binding:"required"`
	Source      string `json:"source" binding:"required"`
	Description string `json:"description"`

	// For group transactions
	GroupID *uuid.UUID `json:"group_id"`
	PaidBy  *uuid.UUID `json:"paid_by"`

	// For linking to planned expense
	PlannedExpenseID *uuid.UUID `json:"planned_expense_id"`
}

type TransactionResponse struct {
	ID          uuid.UUID `json:"id"`
	OwnerType   string    `json:"owner_type"`
	OwnerID     uuid.UUID `json:"owner_id"`
	Type        string    `json:"type"`
	Amount      int64     `json:"amount"`
	Balance     int64     `json:"balance"`
	Category    string    `json:"category"`
	Source      string    `json:"source"`
	Description string    `json:"description"`
	CreatedAt   string    `json:"created_at"`

	GroupID          *uuid.UUID `json:"group_id,omitempty"`
	PaidBy           *uuid.UUID `json:"paid_by,omitempty"`
	PlannedExpenseID *uuid.UUID `json:"planned_expense_id,omitempty"`

	Group *GroupResponse `json:"group,omitempty"`
	Payer *UserResponse  `json:"payer,omitempty"`
}

type TransferToGroupRequest struct {
	GroupID     uuid.UUID `json:"group_id" binding:"required"`
	Amount      int64     `json:"amount" binding:"required,gt=0"`
	Description string    `json:"description"`
}

type PayGroupExpenseRequest struct {
	PlannedExpenseID uuid.UUID `json:"planned_expense_id" binding:"required"`
	ActualPrice      int64     `json:"actual_price" binding:"required,gt=0"`
	Description      string    `json:"description"`
}
