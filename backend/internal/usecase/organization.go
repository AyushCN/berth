package usecase

import (
	"context"
	"errors"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrOrgNotFound    = errors.New("organization not found")
	ErrOrgUnauthorized = errors.New("unauthorized organization action")
)

type OrganizationUsecase struct {
	orgRepo domain.OrganizationRepository
}

func NewOrganizationUsecase(orgRepo domain.OrganizationRepository) *OrganizationUsecase {
	return &OrganizationUsecase{
		orgRepo: orgRepo,
	}
}

func (u *OrganizationUsecase) Create(ctx context.Context, userID uuid.UUID, name string) (*domain.Organization, error) {
	org, err := u.orgRepo.Create(ctx, name)
	if err != nil {
		return nil, err
	}
	// Add the creator as the OWNER
	_, err = u.orgRepo.AddMember(ctx, org.ID, userID, domain.OrgRoleOwner)
	if err != nil {
		return nil, err
	}
	return org, nil
}

func (u *OrganizationUsecase) GetByID(ctx context.Context, userID uuid.UUID, orgID uuid.UUID) (*domain.Organization, error) {
	// Must be a member to view
	_, err := u.orgRepo.GetMember(ctx, orgID, userID)
	if err != nil {
		return nil, ErrOrgUnauthorized
	}
	return u.orgRepo.GetByID(ctx, orgID)
}

func (u *OrganizationUsecase) ListForUser(ctx context.Context, userID uuid.UUID) ([]*domain.Organization, error) {
	return u.orgRepo.ListForUser(ctx, userID)
}

func (u *OrganizationUsecase) AddMember(ctx context.Context, currentUserID uuid.UUID, orgID uuid.UUID, newUserID uuid.UUID, role domain.OrganizationRole) (*domain.OrganizationMember, error) {
	member, err := u.orgRepo.GetMember(ctx, orgID, currentUserID)
	if err != nil || (member.Role != domain.OrgRoleOwner && member.Role != domain.OrgRoleAdmin) {
		return nil, ErrOrgUnauthorized
	}
	return u.orgRepo.AddMember(ctx, orgID, newUserID, role)
}

func (u *OrganizationUsecase) ListMembers(ctx context.Context, currentUserID uuid.UUID, orgID uuid.UUID) ([]*domain.OrganizationMember, error) {
	_, err := u.orgRepo.GetMember(ctx, orgID, currentUserID)
	if err != nil {
		return nil, ErrOrgUnauthorized
	}
	return u.orgRepo.ListMembers(ctx, orgID)
}

func (u *OrganizationUsecase) UpdateMemberRole(ctx context.Context, currentUserID uuid.UUID, orgID uuid.UUID, targetUserID uuid.UUID, role domain.OrganizationRole) error {
	member, err := u.orgRepo.GetMember(ctx, orgID, currentUserID)
	if err != nil || member.Role != domain.OrgRoleOwner {
		return ErrOrgUnauthorized
	}
	return u.orgRepo.UpdateRole(ctx, orgID, targetUserID, role)
}

func (u *OrganizationUsecase) RemoveMember(ctx context.Context, currentUserID uuid.UUID, orgID uuid.UUID, targetUserID uuid.UUID) error {
	member, err := u.orgRepo.GetMember(ctx, orgID, currentUserID)
	if err != nil {
		return ErrOrgUnauthorized
	}
	// Users can remove themselves, but otherwise need ADMIN/OWNER rights
	if currentUserID != targetUserID && member.Role != domain.OrgRoleOwner && member.Role != domain.OrgRoleAdmin {
		return ErrOrgUnauthorized
	}
	return u.orgRepo.RemoveMember(ctx, orgID, targetUserID)
}
