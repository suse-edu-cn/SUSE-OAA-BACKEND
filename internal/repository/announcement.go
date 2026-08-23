package repository

import (
	"context"
	"errors"
	"suseoaa/internal/model"
	"time"

	"gorm.io/gorm"
)

type AnnouncementRepository struct {
	DB *gorm.DB
}

func NewAnnouncementRepository(db *gorm.DB) AnnouncementRepository {
	return AnnouncementRepository{
		DB: db,
	}
}

func (a *AnnouncementRepository) CreateAnnouncement(announcement model.Announcement) (uint64, error) {
	err := a.DB.Model(&model.Announcement{}).Create(&announcement).Error
	if err != nil {
		return 0, errors.New("创建失败")
	}
	return announcement.ID, nil
}

func (a *AnnouncementRepository) UpdateAnnouncement(announcement model.Announcement) error {
	err := a.DB.Model(&model.Announcement{}).
		Where("id = ?", announcement.ID).
		Updates(map[string]any{
			"title":   announcement.Title,
			"content": announcement.Content,
		}).Error
	if err != nil {
		return errors.New("更新失败")
	}
	return nil
}
func (a *AnnouncementRepository) GetDepartmentIDByID(id uint64) (uint64, error) {
	var announcement model.Announcement
	err := a.DB.Model(&model.Announcement{}).Where("id = ?", id).First(&announcement).Error
	if err != nil {
		return 0, err
	}
	return announcement.DepartmentID, nil
}

func (a *AnnouncementRepository) GetAnnouncementInfo(id uint64) (model.AnnouncementInfo, error) {
	var announcement model.Announcement
	err := a.DB.Model(&model.Announcement{}).
		Preload("Department").
		Preload("Publisher").
		Preload("Publisher.Role").
		Where("id = ?", id).First(&announcement).Error
	if err != nil {
		return model.AnnouncementInfo{}, errors.New("获取失败")
	}
	return announcement.ToInfo(), nil
}
func (a *AnnouncementRepository) GetAnnouncementInfoList(isActive bool) (*[]model.AnnouncementInfo, error) {
	var announcementsInfo []model.AnnouncementInfo
	var announcements []model.Announcement
	err := a.DB.Model(&model.Announcement{}).
		Preload("Department").
		Preload("Publisher").
		Preload("Publisher.Role").
		Where("is_active = ? AND  publisher_id IS NOT NULL ", isActive).Find(&announcements).Error
	if err != nil {
		return nil, err
	}
	for _, announcement := range announcements {

		announcementsInfo = append(announcementsInfo, announcement.ToInfo())
	}
	return &announcementsInfo, nil
}
func (a *AnnouncementRepository) PushAnnouncement(
	ctx context.Context,
	id uint64,
	departmentID uint64,
	publisherID uint64,
) error {
	now := time.Now()

	return a.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.Announcement{}).
			Where("id = ? AND department_id = ?", id, departmentID).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return errors.New("公告不存在或不属于该部门")
		}

		return tx.Model(&model.Announcement{}).
			Where("department_id = ? AND (id = ? OR is_active = ?)",
				departmentID, id, true).
			Updates(map[string]any{
				"is_active": gorm.Expr(
					"CASE WHEN id = ? THEN ? ELSE ? END",
					id, true, false,
				),
				"publisher_id": gorm.Expr(
					"CASE WHEN id = ? THEN ? ELSE publisher_id END",
					id, publisherID,
				),
				"published_at": gorm.Expr(
					"CASE WHEN id = ? THEN ? ELSE published_at END",
					id, now,
				),
			}).Error
	})
}
