package repository

import (
	"errors"
	"suseoaa/internal/model"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type TermRepository struct {
	DB *gorm.DB
}

func NewTermRepository(db *gorm.DB) TermRepository {
	return TermRepository{
		DB: db,
	}
}

//申请表

func (t *TermRepository) CreateApplication(application model.Application) error {
	return t.DB.Create(&application).Error
}
func (t *TermRepository) UpdateApplication(application model.Application) error {
	return t.DB.Where("user_id = ?", application.UserID).Updates(&application).Error
}
func (t *TermRepository) GetLatestApplicationByUserID(userID uint64) (*model.Application, error) {
	var application model.Application
	err := t.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&application).Error
	return &application, err
}
func (t *TermRepository) GetApplicationsByUserID(userID uint64) ([]*model.Application, error) {
	var application []*model.Application
	err := t.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&application).Error
	return application, err
}

// 业务周期

func (t *TermRepository) CreateTerm(term model.Term) error {
	if err := t.DB.Create(&term).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return errors.New("创建失败，该年份类型的数据已存在")
		}
		return err
	}
	return nil
}
func (t *TermRepository) UpdateTerm(term model.Term) error {
	tx := t.DB.Model(&model.Term{}).Where("id = ?", term.ID).Updates(map[string]any{
		"title":          term.Title,
		"edit_start_at":  term.EditStartAt,
		"edit_end_at":    term.EditEndAt,
		"query_start_at": term.QueryStartAt,
		"query_end_at":   term.QueryEndAt,
	})

	if tx.Error != nil {
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		return errors.New("数据不存在")
	}

	return nil
}
func (t *TermRepository) GetTermByID(termID uint64) (model.Term, error) {
	var term model.Term
	err := t.DB.Model(&model.Term{}).Where("id = ?", termID).First(&term).Error
	if err != nil {
		return model.Term{}, err
	}
	return term, nil
}
func (t *TermRepository) GetTermList(year uint64, termType string) ([]*model.Term, error) {
	var termList []*model.Term
	tx := t.DB.Model(&model.Term{})
	if year != 0 {
		tx = tx.Where("year = ?", year)
	}
	if termType != "" {
		tx = tx.Where("type = ?", termType)
	}
	err := tx.Find(&termList).Error
	if err != nil {
		return nil, err
	}
	return termList, nil
}

func (t *TermRepository) GetTermMap() (map[uint64]model.Term, error) {
	termMap := make(map[uint64]model.Term)
	var termList []model.Term
	err := t.DB.Model(&model.Term{}).Find(&termList).Error
	if err != nil {
		return nil, err
	}
	for _, term := range termList {
		termMap[term.ID] = term
	}
	return termMap, nil
}

//面试官

func (t *TermRepository) CreateInterviewers(interviewer []model.Interviewer) error {
	return t.DB.Create(&interviewer).Error
}
func (t *TermRepository) UpdateInterviewers(interviewers model.Interviewer) error {
	return t.DB.Where("id = ?", interviewers.ID).Updates(&interviewers).Error
}

func (t *TermRepository) GetInterviewerListByTermID(termID uint64) ([]model.Interviewer, error) {
	var interviewerList []model.Interviewer
	tx := t.DB.Model(&model.Interviewer{})
	if termID != 0 {
		tx = tx.Where("term_id = ?", termID)
	}
	err := tx.Find(&interviewerList).Error
	if err != nil {
		return nil, err
	}
	return interviewerList, nil
}

//招新换届的历史
