package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Group struct {
	BaseModel
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	Balance     int64     `gorm:"default:0" json:"balance"`
	CreatedBy   uuid.UUID `gorm:"not null" json:"created_by"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`

	// Relationships
	Members         []UserGroup      `gorm:"foreignKey:GroupID" json:"members"`
	Transactions    []Transaction    `gorm:"foreignKey:GroupID" json:"-"`
	PlannedExpenses []PlannedExpense `gorm:"foreignKey:GroupID" json:"-"`
	AuditLogs       []AuditLog       `gorm:"foreignKey:GroupID" json:"-"`
}

type UserGroup struct {
	BaseModel
	UserID   uuid.UUID `gorm:"not null;index" json:"user_id"`
	GroupID  uuid.UUID `gorm:"not null;index" json:"group_id"`
	Role     string    `gorm:"not null;default:'member'" json:"role"`   // member, manager
	Status   string    `gorm:"not null;default:'active'" json:"status"` // pending, active, rejected, left
	JoinedAt time.Time `json:"joined_at"`

	// Relationships
	User  User  `gorm:"foreignKey:UserID" json:"user"`
	Group Group `gorm:"foreignKey:GroupID" json:"group"`
}

func (g *Group) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}

func (ug *UserGroup) BeforeCreate(tx *gorm.DB) error {
	if ug.ID == uuid.Nil {
		ug.ID = uuid.New()
	}
	if ug.JoinedAt.IsZero() {
		ug.JoinedAt = time.Now()
	}
	return nil
}
