package dto

import "github.com/google/uuid"

type CreateGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type GroupResponse struct {
	ID          uuid.UUID        `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Balance     int64            `json:"balance"`
	CreatedBy   uuid.UUID        `json:"created_by"`
	IsActive    bool             `json:"is_active"`
	CreatedAt   string           `json:"created_at"`
	Members     []MemberResponse `json:"members"`
}

type MemberResponse struct {
	ID       uuid.UUID    `json:"id"`
	UserID   uuid.UUID    `json:"user_id"`
	GroupID  uuid.UUID    `json:"group_id"`
	Role     string       `json:"role"`
	Status   string       `json:"status"`
	JoinedAt string       `json:"joined_at"`
	User     UserResponse `json:"user"`
}

type InviteMemberRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Role        string `json:"role" binding:"required,oneof=member manager"`
}

type UpdateMemberRoleRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	Role   string    `json:"role" binding:"required,oneof=member manager"`
}

type GroupInvitationResponse struct {
	ID        uuid.UUID    `json:"id"`
	GroupID   uuid.UUID    `json:"group_id"`
	GroupName string       `json:"group_name"`
	InvitedBy UserResponse `json:"invited_by"`
	Role      string       `json:"role"`
	Status    string       `json:"status"`
	CreatedAt string       `json:"created_at"`
}
