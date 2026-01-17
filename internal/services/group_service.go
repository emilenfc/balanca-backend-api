package services

import (
	"balanca/internal/dto"
	"balanca/internal/models"
	"balanca/internal/repositories"
	"balanca/pkg/errors"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type GroupService interface {
	CreateGroup(userID uuid.UUID, req dto.CreateGroupRequest) (*dto.GroupResponse, error)
	GetGroup(userID, groupID uuid.UUID) (*dto.GroupResponse, error)
	GetGroups(userID uuid.UUID) ([]dto.GroupResponse, error)
	InviteMember(userID, groupID uuid.UUID, req dto.InviteMemberRequest) error
	AcceptInvitation(userID, invitationID uuid.UUID) error
	RejectInvitation(userID, invitationID uuid.UUID) error
	UpdateMemberRole(userID, groupID uuid.UUID, req dto.UpdateMemberRoleRequest) error
	RemoveMember(userID, groupID, targetUserID uuid.UUID) error
	GetPendingInvitations(userID uuid.UUID) ([]dto.GroupInvitationResponse, error)
	LeaveGroup(userID, groupID uuid.UUID) error
	DeleteGroup(userID, groupID uuid.UUID) error
}

type groupService struct {
	groupRepo repositories.GroupRepository
	userRepo  repositories.UserRepository
	auditRepo repositories.AuditLogRepository
	db        *gorm.DB
}

func NewGroupService(
	groupRepo repositories.GroupRepository,
	userRepo repositories.UserRepository,
	auditRepo repositories.AuditLogRepository,
	db *gorm.DB,
) GroupService {
	return &groupService{
		groupRepo: groupRepo,
		userRepo:  userRepo,
		auditRepo: auditRepo,
		db:        db,
	}
}

func (s *groupService) CreateGroup(userID uuid.UUID, req dto.CreateGroupRequest) (*dto.GroupResponse, error) {
	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create group
	group := &models.Group{
		Name:        req.Name,
		Description: req.Description,
		Balance:     0,
		CreatedBy:   userID,
		IsActive:    true,
	}

	if err := tx.Create(group).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to create group")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create group"}
	}

	// Add creator as manager
	userGroup := &models.UserGroup{
		UserID:  userID,
		GroupID: group.ID,
		Role:    "manager",
		Status:  "active",
	}

	if err := tx.Create(userGroup).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to add creator to group")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create group"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "group",
		EntityID:    group.ID,
		Action:      "create",
		PerformedBy: userID,
		GroupID:     &group.ID,
	}

	if err := tx.Create(auditLog).Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to create audit log")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create group"}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		log.Error().Err(err).Msg("Failed to commit transaction")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to create group"}
	}

	// Get full group data
	fullGroup, err := s.groupRepo.FindByID(group.ID)
	if err != nil {
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get group data"}
	}

	// Build response
	var members []dto.MemberResponse
	for _, member := range fullGroup.Members {
		members = append(members, dto.MemberResponse{
			ID:       member.ID,
			UserID:   member.UserID,
			GroupID:  member.GroupID,
			Role:     member.Role,
			Status:   member.Status,
			JoinedAt: member.JoinedAt.Format(time.RFC3339),
			User: dto.UserResponse{
				ID:          member.User.ID,
				PhoneNumber: member.User.PhoneNumber,
				Email:       member.User.Email,
				FirstName:   member.User.FirstName,
				LastName:    member.User.LastName,
				Balance:     member.User.Balance,
				IsActive:    member.User.IsActive,
				CreatedAt:   member.User.CreatedAt.Format(time.RFC3339),
			},
		})
	}

	return &dto.GroupResponse{
		ID:          fullGroup.ID,
		Name:        fullGroup.Name,
		Description: fullGroup.Description,
		Balance:     fullGroup.Balance,
		CreatedBy:   fullGroup.CreatedBy,
		IsActive:    fullGroup.IsActive,
		CreatedAt:   fullGroup.CreatedAt.Format(time.RFC3339),
		Members:     members,
	}, nil
}

func (s *groupService) GetGroup(userID, groupID uuid.UUID) (*dto.GroupResponse, error) {
	// Check if user is a member of the group
	userGroup, err := s.groupRepo.FindByUserAndGroup(userID, groupID)
	if err != nil || userGroup.Status != "active" {
		return nil, &errors.AppError{Code: "FORBIDDEN", Message: "You are not a member of this group"}
	}

	group, err := s.groupRepo.FindByID(groupID)
	if err != nil {
		return nil, &errors.AppError{Code: "GROUP_NOT_FOUND", Message: "Group not found"}
	}

	// Build response
	var members []dto.MemberResponse
	for _, member := range group.Members {
		members = append(members, dto.MemberResponse{
			ID:       member.ID,
			UserID:   member.UserID,
			GroupID:  member.GroupID,
			Role:     member.Role,
			Status:   member.Status,
			JoinedAt: member.JoinedAt.Format(time.RFC3339),
			User: dto.UserResponse{
				ID:          member.User.ID,
				PhoneNumber: member.User.PhoneNumber,
				Email:       member.User.Email,
				FirstName:   member.User.FirstName,
				LastName:    member.User.LastName,
				Balance:     member.User.Balance,
				IsActive:    member.User.IsActive,
				CreatedAt:   member.User.CreatedAt.Format(time.RFC3339),
			},
		})
	}

	return &dto.GroupResponse{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		Balance:     group.Balance,
		CreatedBy:   group.CreatedBy,
		IsActive:    group.IsActive,
		CreatedAt:   group.CreatedAt.Format(time.RFC3339),
		Members:     members,
	}, nil
}

func (s *groupService) GetGroups(userID uuid.UUID) ([]dto.GroupResponse, error) {
	groups, err := s.groupRepo.FindUserGroups(userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user groups")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get groups"}
	}

	var response []dto.GroupResponse
	for _, group := range groups {
		var members []dto.MemberResponse
		for _, member := range group.Members {
			members = append(members, dto.MemberResponse{
				ID:       member.ID,
				UserID:   member.UserID,
				GroupID:  member.GroupID,
				Role:     member.Role,
				Status:   member.Status,
				JoinedAt: member.JoinedAt.Format(time.RFC3339),
				User: dto.UserResponse{
					ID:          member.User.ID,
					PhoneNumber: member.User.PhoneNumber,
					Email:       member.User.Email,
					FirstName:   member.User.FirstName,
					LastName:    member.User.LastName,
					Balance:     member.User.Balance,
					IsActive:    member.User.IsActive,
					CreatedAt:   member.User.CreatedAt.Format(time.RFC3339),
				},
			})
		}

		response = append(response, dto.GroupResponse{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Balance:     group.Balance,
			CreatedBy:   group.CreatedBy,
			IsActive:    group.IsActive,
			CreatedAt:   group.CreatedAt.Format(time.RFC3339),
			Members:     members,
		})
	}

	return response, nil
}

func (s *groupService) InviteMember(userID, groupID uuid.UUID, req dto.InviteMemberRequest) error {
	// Check if inviter is a manager in the group
	inviterGroup, err := s.groupRepo.FindByUserAndGroup(userID, groupID)
	if err != nil || inviterGroup.Status != "active" || inviterGroup.Role != "manager" {
		return &errors.AppError{Code: "FORBIDDEN", Message: "Only managers can invite members"}
	}

	// Find user by phone number
	userToInvite, err := s.userRepo.FindByPhoneNumber(req.PhoneNumber)
	if err != nil {
		return &errors.AppError{Code: "USER_NOT_FOUND", Message: "User not found"}
	}

	// Check if user is already a member
	existingMembership, _ := s.groupRepo.FindByUserAndGroup(userToInvite.ID, groupID)
	if existingMembership != nil {
		if existingMembership.Status == "active" {
			return &errors.AppError{Code: "ALREADY_MEMBER", Message: "User is already a member of this group"}
		}
		if existingMembership.Status == "pending" {
			return &errors.AppError{Code: "ALREADY_INVITED", Message: "User has already been invited"}
		}
	}

	// Create invitation
	invitation := &models.UserGroup{
		UserID:  userToInvite.ID,
		GroupID: groupID,
		Role:    req.Role,
		Status:  "pending",
	}

	if err := s.groupRepo.AddMember(invitation); err != nil {
		log.Error().Err(err).Msg("Failed to invite member")
		return &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to invite member"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "user_group",
		EntityID:    invitation.ID,
		Action:      "invite",
		Changes:     map[string]interface{}{"role": req.Role},
		PerformedBy: userID,
		GroupID:     &groupID,
	}

	if err := s.auditRepo.Create(auditLog); err != nil {
		log.Error().Err(err).Msg("Failed to create audit log")
		// Don't return error for audit log failure
	}

	// TODO: Send notification to invited user

	return nil
}

func (s *groupService) AcceptInvitation(userID, invitationID uuid.UUID) error {
	// Get the invitation
	invitation, err := s.groupRepo.FindByUserAndGroup(userID, invitationID)
	if err != nil {
		return &errors.AppError{Code: "INVITATION_NOT_FOUND", Message: "Invitation not found"}
	}

	if invitation.Status != "pending" {
		return &errors.AppError{Code: "INVALID_INVITATION", Message: "Invitation is not pending"}
	}

	// Update invitation status
	invitation.Status = "active"
	if err := s.groupRepo.UpdateMember(invitation); err != nil {
		log.Error().Err(err).Msg("Failed to accept invitation")
		return &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to accept invitation"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "user_group",
		EntityID:    invitation.ID,
		Action:      "accept_invitation",
		PerformedBy: userID,
		GroupID:     &invitation.GroupID,
	}

	if err := s.auditRepo.Create(auditLog); err != nil {
		log.Error().Err(err).Msg("Failed to create audit log")
	}

	return nil
}

func (s *groupService) RejectInvitation(userID, invitationID uuid.UUID) error {
	// Get the invitation
	invitation, err := s.groupRepo.FindByUserAndGroup(userID, invitationID)
	if err != nil {
		return &errors.AppError{Code: "INVITATION_NOT_FOUND", Message: "Invitation not found"}
	}

	if invitation.Status != "pending" {
		return &errors.AppError{Code: "INVALID_INVITATION", Message: "Invitation is not pending"}
	}

	// Update invitation status
	invitation.Status = "rejected"
	if err := s.groupRepo.UpdateMember(invitation); err != nil {
		log.Error().Err(err).Msg("Failed to reject invitation")
		return &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to reject invitation"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "user_group",
		EntityID:    invitation.ID,
		Action:      "reject_invitation",
		PerformedBy: userID,
		GroupID:     &invitation.GroupID,
	}

	if err := s.auditRepo.Create(auditLog); err != nil {
		log.Error().Err(err).Msg("Failed to create audit log")
	}

	return nil
}

func (s *groupService) UpdateMemberRole(userID, groupID uuid.UUID, req dto.UpdateMemberRoleRequest) error {
	// Check if user is a manager
	userGroup, err := s.groupRepo.FindByUserAndGroup(userID, groupID)
	if err != nil || userGroup.Status != "active" || userGroup.Role != "manager" {
		return &errors.AppError{Code: "FORBIDDEN", Message: "Only managers can update member roles"}
	}

	// Get target user's membership
	targetUserGroup, err := s.groupRepo.FindByUserAndGroup(req.UserID, groupID)
	if err != nil || targetUserGroup.Status != "active" {
		return &errors.AppError{Code: "MEMBER_NOT_FOUND", Message: "Member not found"}
	}

	// Update role
	oldRole := targetUserGroup.Role
	targetUserGroup.Role = req.Role

	if err := s.groupRepo.UpdateMember(targetUserGroup); err != nil {
		log.Error().Err(err).Msg("Failed to update member role")
		return &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to update member role"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "user_group",
		EntityID:    targetUserGroup.ID,
		Action:      "update_role",
		Changes:     map[string]interface{}{"old_role": oldRole, "new_role": req.Role},
		PerformedBy: userID,
		GroupID:     &groupID,
	}

	if err := s.auditRepo.Create(auditLog); err != nil {
		log.Error().Err(err).Msg("Failed to create audit log")
	}

	return nil
}

func (s *groupService) RemoveMember(userID, groupID, targetUserID uuid.UUID) error {
	// Check if user is a manager
	userGroup, err := s.groupRepo.FindByUserAndGroup(userID, groupID)
	if err != nil || userGroup.Status != "active" || userGroup.Role != "manager" {
		return &errors.AppError{Code: "FORBIDDEN", Message: "Only managers can remove members"}
	}

	// Cannot remove yourself
	if userID == targetUserID {
		return &errors.AppError{Code: "FORBIDDEN", Message: "Cannot remove yourself from group"}
	}

	// Remove member
	if err := s.groupRepo.RemoveMember(targetUserID, groupID); err != nil {
		log.Error().Err(err).Msg("Failed to remove member")
		return &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to remove member"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "user_group",
		EntityID:    targetUserID,
		Action:      "remove_member",
		PerformedBy: userID,
		GroupID:     &groupID,
	}

	if err := s.auditRepo.Create(auditLog); err != nil {
		log.Error().Err(err).Msg("Failed to create audit log")
	}

	return nil
}

func (s *groupService) GetPendingInvitations(userID uuid.UUID) ([]dto.GroupInvitationResponse, error) {
	invitations, err := s.groupRepo.FindPendingInvitations(userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get pending invitations")
		return nil, &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to get invitations"}
	}

	var response []dto.GroupInvitationResponse
	for _, invitation := range invitations {
		// Get group creator info
		creator, _ := s.userRepo.FindByID(invitation.Group.CreatedBy)

		response = append(response, dto.GroupInvitationResponse{
			ID:        invitation.ID,
			GroupID:   invitation.GroupID,
			GroupName: invitation.Group.Name,
			InvitedBy: dto.UserResponse{
				ID:          creator.ID,
				PhoneNumber: creator.PhoneNumber,
				Email:       creator.Email,
				FirstName:   creator.FirstName,
				LastName:    creator.LastName,
				Balance:     creator.Balance,
				IsActive:    creator.IsActive,
				CreatedAt:   creator.CreatedAt.Format(time.RFC3339),
			},
			Role:      invitation.Role,
			Status:    invitation.Status,
			CreatedAt: invitation.CreatedAt.Format(time.RFC3339),
		})
	}

	return response, nil
}

func (s *groupService) LeaveGroup(userID, groupID uuid.UUID) error {
	// Check if user is a member
	userGroup, err := s.groupRepo.FindByUserAndGroup(userID, groupID)
	if err != nil || userGroup.Status != "active" {
		return &errors.AppError{Code: "NOT_MEMBER", Message: "You are not a member of this group"}
	}

	// Check if user is the last manager
	if userGroup.Role == "manager" {
		members, err := s.groupRepo.FindMembers(groupID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get group members")
			return &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to leave group"}
		}

		managerCount := 0
		for _, member := range members {
			if member.Status == "active" && member.Role == "manager" {
				managerCount++
			}
		}

		if managerCount <= 1 {
			return &errors.AppError{Code: "LAST_MANAGER", Message: "Cannot leave as the last manager. Promote someone else first or delete the group."}
		}
	}

	// Remove user from group
	if err := s.groupRepo.RemoveMember(userID, groupID); err != nil {
		log.Error().Err(err).Msg("Failed to leave group")
		return &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to leave group"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "user_group",
		EntityID:    userGroup.ID,
		Action:      "leave_group",
		PerformedBy: userID,
		GroupID:     &groupID,
	}

	if err := s.auditRepo.Create(auditLog); err != nil {
		log.Error().Err(err).Msg("Failed to create audit log")
	}

	return nil
}

func (s *groupService) DeleteGroup(userID, groupID uuid.UUID) error {
	// Check if user is a manager
	userGroup, err := s.groupRepo.FindByUserAndGroup(userID, groupID)
	if err != nil || userGroup.Status != "active" || userGroup.Role != "manager" {
		return &errors.AppError{Code: "FORBIDDEN", Message: "Only managers can delete the group"}
	}

	// Delete group
	if err := s.groupRepo.Delete(groupID); err != nil {
		log.Error().Err(err).Msg("Failed to delete group")
		return &errors.AppError{Code: "SERVER_ERROR", Message: "Failed to delete group"}
	}

	// Create audit log
	auditLog := &models.AuditLog{
		Entity:      "group",
		EntityID:    groupID,
		Action:      "delete",
		PerformedBy: userID,
		GroupID:     &groupID,
	}

	if err := s.auditRepo.Create(auditLog); err != nil {
		log.Error().Err(err).Msg("Failed to create audit log")
	}

	return nil
}
