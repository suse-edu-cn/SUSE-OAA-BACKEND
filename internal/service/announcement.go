package service

import (
	"context"
	"errors"
	"fmt"
	"suseoaa/internal/model"
	"suseoaa/internal/repository"
)

type AnnouncementService struct {
	AnnouncementRepo repository.AnnouncementRepository
	DepartmentRepo   repository.DepartmentRepository
	RoleRepo         repository.RoleRepository
	UserRepo         repository.UserRepository
	FileService      FileService
}

func NewAnnouncementService(
	announcementRepo repository.AnnouncementRepository,
	departmentRepo repository.DepartmentRepository,
	roleRepo repository.RoleRepository,
	userRepo repository.UserRepository,
	fileService FileService,
) AnnouncementService {
	return AnnouncementService{
		AnnouncementRepo: announcementRepo,
		DepartmentRepo:   departmentRepo,
		RoleRepo:         roleRepo,
		UserRepo:         userRepo,
		FileService:      fileService,
	}
}

func (a *AnnouncementService) check(userID uint64, departmentID uint64) error {

	level, userDepartment, err := a.UserRepo.GetActiveRoleLevelAndDepartment(userID)
	if err != nil {
		return err
	}
	department, err := a.DepartmentRepo.GetDepartmentByID(departmentID)
	if err != nil {
		return err
	}
	if level >= 80 {
		return nil
	}
	if level < 50 || department.Name != userDepartment {
		return errors.New("权限不够")
	}
	return nil
}

func (a *AnnouncementService) CreateAnnouncement(userID uint64, announcement model.Announcement) (map[string]uint64, error) {
	err := a.check(userID, announcement.DepartmentID)
	if err != nil {
		return nil, err
	}
	announcementID, err := a.AnnouncementRepo.CreateAnnouncement(announcement)
	if err != nil {
		return nil, err
	}
	return map[string]uint64{
		"announcement_id": announcementID,
	}, nil
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

func (a *AnnouncementService) GetAnnouncementInfoList(ctx context.Context, id uint64, status string, isContent bool) (*[]model.AnnouncementInfo, error) {
	announcementList, err := a.AnnouncementRepo.GetAnnouncementInfoListByRole(id, status)
	if err != nil {
		return nil, err
	}
	if announcementList != nil {
		list := *announcementList
		for i := range list {
			if isContent {
				content, err := a.FileService.ReplaceMinIOLinks(ctx, list[i].Content)
				if err != nil {
					_ = fmt.Errorf(err.Error())
				}
				list[i].Content = content
			} else {
				list[i].Content = ""
			}
		}
	}
	return announcementList, nil
}

func (a *AnnouncementService) GetAnnouncementInfo(ctx context.Context, userID uint64, announcementID uint64) (model.AnnouncementInfo, error) {
	announcement, err := a.AnnouncementRepo.GetAnnouncementInfo(announcementID, userID)
	if err != nil {
		return model.AnnouncementInfo{}, err
	}
	content, err := a.FileService.ReplaceMinIOLinks(ctx, announcement.Content)
	if err != nil {
		return model.AnnouncementInfo{}, err
	}
	announcement.Content = content
	return announcement, nil
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
