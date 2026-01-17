package repositories

import (
	"balanca/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GroupRepository interface {
	Create(group *models.Group) error
	FindByID(id uuid.UUID) (*models.Group, error)
	Update(group *models.Group) error
	Delete(id uuid.UUID) error
	FindUserGroups(userID uuid.UUID) ([]models.Group, error)
	FindByUserAndGroup(userID, groupID uuid.UUID) (*models.UserGroup, error)
	AddMember(userGroup *models.UserGroup) error
	RemoveMember(userID, groupID uuid.UUID) error
	UpdateMember(userGroup *models.UserGroup) error
	FindMembers(groupID uuid.UUID) ([]models.UserGroup, error)
	FindPendingInvitations(userID uuid.UUID) ([]models.UserGroup, error)
	
}

type groupRepository struct {
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &groupRepository{db: db}
}

func (r *groupRepository) Create(group *models.Group) error {
	return r.db.Create(group).Error
}

func (r *groupRepository) FindByID(id uuid.UUID) (*models.Group, error) {
	var group models.Group
	err := r.db.Preload("Members.User").Where("id = ?", id).First(&group).Error
	return &group, err
}

func (r *groupRepository) Update(group *models.Group) error {
	return r.db.Save(group).Error
}

func (r *groupRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Group{}, "id = ?", id).Error
}

func (r *groupRepository) FindUserGroups(userID uuid.UUID) ([]models.Group, error) {
	var groups []models.Group
	err := r.db.Joins("JOIN user_groups ON user_groups.group_id = groups.id").
		Where("user_groups.user_id = ? AND user_groups.status = ?", userID, "active").
		Preload("Members.User").
		Find(&groups).Error
	return groups, err
}

func (r *groupRepository) FindByUserAndGroup(userID, groupID uuid.UUID) (*models.UserGroup, error) {
	var userGroup models.UserGroup
	err := r.db.Preload("User").Preload("Group").
		Where("user_id = ? AND group_id = ?", userID, groupID).
		First(&userGroup).Error
	return &userGroup, err
}

func (r *groupRepository) AddMember(userGroup *models.UserGroup) error {
	return r.db.Create(userGroup).Error
}

func (r *groupRepository) RemoveMember(userID, groupID uuid.UUID) error {
	return r.db.Delete(&models.UserGroup{}, "user_id = ? AND group_id = ?", userID, groupID).Error
}

func (r *groupRepository) UpdateMember(userGroup *models.UserGroup) error {
	return r.db.Save(userGroup).Error
}

func (r *groupRepository) FindMembers(groupID uuid.UUID) ([]models.UserGroup, error) {
	var members []models.UserGroup
	err := r.db.Preload("User").Where("group_id = ?", groupID).Find(&members).Error
	return members, err
}

func (r *groupRepository) FindPendingInvitations(userID uuid.UUID) ([]models.UserGroup, error) {
	var invitations []models.UserGroup
	err := r.db.Preload("Group").Where("user_id = ? AND status = ?", userID, "pending").Find(&invitations).Error
	return invitations, err
}