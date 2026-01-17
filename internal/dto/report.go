package dto

import (
	"time"

	"github.com/google/uuid"
)

type DateRangeRequest struct {
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required"`
}

type MonthlyReportRequest struct {
	Year  int `json:"year" binding:"required"`
	Month int `json:"month" binding:"required,min=1,max=12"`
}

type MonthlyReportResponse struct {
	Month           string                `json:"month"`
	Year            int                   `json:"year"`
	TotalIncome     int64                 `json:"total_income"`
	TotalExpenses   int64                 `json:"total_expenses"`
	NetBalance      int64                 `json:"net_balance"`
	StartingBalance int64                 `json:"starting_balance"`
	EndingBalance   int64                 `json:"ending_balance"`
	Transactions    []TransactionResponse `json:"transactions"`
	Categories      []CategorySummary     `json:"categories"`
	Sources         []SourceSummary       `json:"sources"`
}

type CategorySummary struct {
	Category   string  `json:"category"`
	Amount     int64   `json:"amount"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type SourceSummary struct {
	Source     string  `json:"source"`
	Amount     int64   `json:"amount"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type GroupReportResponse struct {
	GroupID         uuid.UUID              `json:"group_id"`
	GroupName       string                 `json:"group_name"`
	Period          string                 `json:"period"`
	TotalIncome     int64                  `json:"total_income"`
	TotalExpenses   int64                  `json:"total_expenses"`
	NetBalance      int64                  `json:"net_balance"`
	StartingBalance int64                  `json:"starting_balance"`
	EndingBalance   int64                  `json:"ending_balance"`
	Members         []MemberContribution   `json:"members"`
	ExternalSources []ExternalContribution `json:"external_sources"`
	Expenses        []GroupExpenseSummary  `json:"expenses"`
}

type MemberContribution struct {
	UserID     uuid.UUID `json:"user_id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Amount     int64     `json:"amount"`
	Percentage float64   `json:"percentage"`
}

type ExternalContribution struct {
	Source     string  `json:"source"`
	Amount     int64   `json:"amount"`
	Percentage float64 `json:"percentage"`
}

type GroupExpenseSummary struct {
	Category  string    `json:"category"`
	Amount    int64     `json:"amount"`
	Count     int       `json:"count"`
	PaidBy    uuid.UUID `json:"paid_by"`
	PayerName string    `json:"payer_name"`
}
