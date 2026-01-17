package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}
type User struct {
	BaseModel
	PhoneNumber  string `gorm:"uniqueIndex;not null" json:"phone_number"`
	Email        string `gorm:"uniqueIndex" json:"email"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	PasswordHash string `gorm:"not null" json:"-"`
	Balance      int64  `gorm:"default:0" json:"balance"` // in cents
	IsActive     bool   `gorm:"default:true" json:"is_active"`

	// Relationships
	Groups          []UserGroup      `gorm:"foreignKey:UserID" json:"-"`
	Transactions    []Transaction    `gorm:"foreignKey:UserID" json:"-"`
	PlannedExpenses []PlannedExpense `gorm:"foreignKey:UserID" json:"-"`
	Notifications   []Notification   `gorm:"foreignKey:UserID" json:"-"`
	AuditLogs       []AuditLog       `gorm:"foreignKey:PerformedBy" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
