package service

import (
	"context"
	"errors"
	"suseoaa/internal/model"
	"suseoaa/internal/repository"
)

type AnnouncementService struct {
	AnnouncementRepo repository.AnnouncementRepository
	DepartmentRepo   repository.DepartmentRepository
	RoleRepo         repository.RoleRepository
	UserRepo         repository.UserRepository
}

func NewAnnouncementService(
	announcementRepo repository.AnnouncementRepository,
	departmentRepo repository.DepartmentRepository,
	roleRepo repository.RoleRepository,
	userRepo repository.UserRepository,
) AnnouncementService {
	return AnnouncementService{
		AnnouncementRepo: announcementRepo,
		DepartmentRepo:   departmentRepo,
		RoleRepo:         roleRepo,
		UserRepo:         userRepo,
	}
}

func (a *AnnouncementService) check(userID uint64, departmentID uint64) error {

	level, userDepartment, err := a.UserRepo.GetRoleLevelAndDepartment(userID)
	if err != nil {
		return err
	}
	if level >= 80 {
		return nil
	}
	department, err := a.DepartmentRepo.GetDepartmentByID(departmentID)
	if err != nil {
		return err
	}
	if level < 50 || department.Name != userDepartment {
		return errors.New("权限不够")
	}
	return nil
}

func (a *AnnouncementService) CreateAnnouncement(userID uint64, announcement model.Announcement) (uint64, error) {
	err := a.check(userID, announcement.DepartmentID)
	if err != nil {
		return 0, err
	}
	return a.AnnouncementRepo.CreateAnnouncement(announcement)
}
func (a *AnnouncementService) UpdateAnnouncement(userID uint64, announcement model.Announcement) error {
	department, err := a.AnnouncementRepo.GetDepartmentIDByID(announcement.ID)
	if err != nil {
		return err
	}
	err = a.check(userID, department)
	if err != nil {
		return err
	}
	return a.AnnouncementRepo.UpdateAnnouncement(announcement)
}

func (a *AnnouncementService) PushAnnouncement(ctx context.Context, announcementID uint64,
	pushedID uint64) error {
	departmentID, err := a.AnnouncementRepo.GetDepartmentIDByID(announcementID)
	if err != nil {
		return err
	}
	err = a.check(pushedID, departmentID)
	if err != nil {
		return err
	}
	err = a.AnnouncementRepo.PushAnnouncement(ctx, announcementID, departmentID, pushedID)
	return err
}

func (a *AnnouncementService) GetAnnouncementInfoList(id uint64, status string) (*[]model.AnnouncementInfo, error) {
	return a.AnnouncementRepo.GetAnnouncementInfoListByRole(id, status)
}

func (a *AnnouncementService) DeleteAnnouncement(id uint64, userID uint64) error {
	departmentID, err := a.AnnouncementRepo.GetDepartmentIDByID(id)
	if err != nil {
		return err
	}
	err = a.check(userID, departmentID)
	if err != nil {
		return err
	}
	return a.AnnouncementRepo.DeleteAnnouncement(id)
}
