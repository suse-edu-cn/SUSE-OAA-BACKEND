package repository

import (
	"context"
	"errors"
	"suseoaa/internal/model"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	return t.DB.Where("user_id = ? AND term_id = ?", application.UserID, application.TermID).Updates(&application).Error
}
func (t *TermRepository) GetLatestApplicationByUserID(userID uint64) (*model.Application, error) {
	var application model.Application
	err := t.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC, id DESC").
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
func (t *TermRepository) GetApplicationsByTermIDAndDepartmentID(termID uint64, departmentID uint64) ([]*model.Application, error) {
	var application []*model.Application
	tx := t.DB
	if departmentID != 0 {
		tx = tx.Where("first_department_id = ? OR second_department_id = ?", departmentID, departmentID)
	}
	if termID != 0 {
		tx = tx.Where("term_id = ?", termID)
	}
	err := tx.Order("created_at DESC").Find(&application).Error
	return application, err
}
func (t *TermRepository) GetApplicationByID(applicationID uint64) (*model.Application, error) {
	var application model.Application
	err := t.DB.Model(&model.Application{}).Where("id = ?", applicationID).First(&application).Error
	return &application, err

}

func (t *TermRepository) DeleteApplication(applicationID uint64) error {
	result := t.DB.Delete(&model.Application{}, applicationID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("数据不存在")
	}
	return nil
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
		"title":            term.Title,
		"edit_start_at":    term.EditStartAt,
		"edit_end_at":      term.EditEndAt,
		"query_start_at":   term.QueryStartAt,
		"query_end_at":     term.QueryEndAt,
		"execute_after_at": term.ExecuteAfterAt,
	})

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
func (t *TermRepository) GetTermByID(termID uint64) (model.Term, error) {
	var term model.Term
	err := t.DB.Model(&model.Term{}).Where("id = ?", termID).First(&term).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Term{}, errors.New("term 不存在")
		}
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
func (t *TermRepository) DeleteTerm(termID uint64) error {
	return t.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.Term{}).Where("id = ?", termID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return errors.New("term 不存在")
		}

		if err := tx.Delete(&model.InterviewResult{}, "term_id = ?", termID).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Application{}, "term_id = ?", termID).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Interviewer{}, "term_id = ?", termID).Error; err != nil {
			return err
		}

		return tx.Delete(&model.Term{}, termID).Error
	})
}

//面试官

func (t *TermRepository) CreateInterviewers(interviewer []model.Interviewer) error {
	return t.DB.Create(&interviewer).Error
}
func (t *TermRepository) UpdateInterviewers(interviewer model.Interviewer) error {
	var existing model.Interviewer
	if err := t.DB.Select("id").First(&existing, interviewer.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("面试官记录不存在")
		}
		return err
	}

	tx := t.DB.Model(&model.Interviewer{}).Where("id = ?", interviewer.ID).Updates(&interviewer)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
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
func (t *TermRepository) GetInterviewerByID(id uint64) (model.Interviewer, error) {
	var interviewer model.Interviewer
	err := t.DB.Where("id = ?", id).First(&interviewer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Interviewer{}, errors.New("面试官记录不存在")
		}
		return model.Interviewer{}, err
	}
	return interviewer, nil
}

func (t *TermRepository) DeleteInterviewer(id uint64) error {
	result := t.DB.Delete(&model.Interviewer{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("面试官记录不存在")
	}
	return nil
}

func (t *TermRepository) CheckTermIsHave(termID uint64) error {
	var count int64
	err := t.DB.Model(&model.Term{}).Where("id = ?", termID).Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("term 不存在")
	}
	return nil

}

func (t *TermRepository) GetInterviewerListByScope(termID uint64, departmentID uint64) ([]model.Interviewer, error) {
	var interviewerList []model.Interviewer
	tx := t.DB.Model(&model.Interviewer{})
	if termID != 0 {
		tx = tx.Where("term_id = ?", termID)
	}
	if departmentID != 0 {
		tx = tx.Where("department_id = ?", departmentID)
	}
	err := tx.Find(&interviewerList).Error
	if err != nil {
		return nil, err
	}
	return interviewerList, nil
}

//招新换届的历史

//面试结果

func (t *TermRepository) CreateInterviewResult(interviewResult model.InterviewResult) error {
	return t.DB.Create(&interviewResult).Error
}

func (t *TermRepository) UpdateInterviewResult(interviewResult model.InterviewResult) error {
	var existing model.InterviewResult
	if err := t.DB.Select("id").First(&existing, interviewResult.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("面试结果不存在")
		}
		return err
	}

	return t.DB.
		Model(&model.InterviewResult{}).
		Where("id = ?", interviewResult.ID).
		Select("decision", "result_department_id", "result_role_id", "operator_id", "remark").
		Updates(&interviewResult).Error
}
func (t *TermRepository) GetInterviewResultByID(id uint64) (model.InterviewResult, error) {
	var interviewResult model.InterviewResult
	err := t.DB.Where("id = ?", id).First(&interviewResult).Error
	if err != nil {
		return model.InterviewResult{}, err
	}
	return interviewResult, nil
}

func (t *TermRepository) GetInterviewResultList(termID uint64) ([]model.InterviewResult, error) {
	var interviewResults []model.InterviewResult
	tx := t.DB.Model(&model.InterviewResult{})
	if termID != 0 {
		tx = tx.Where("term_id = ?", termID)
	}
	err := tx.Order("created_at DESC, id DESC").Find(&interviewResults).Error
	if err != nil {
		return nil, err
	}
	return interviewResults, nil
}

//----------------------------
//面试结果执行

func (t *TermRepository) GetExecutableTerms(ctx context.Context, now time.Time) ([]model.Term, error) {
	var terms []model.Term
	err := t.DB.WithContext(ctx).
		Where("is_executed = ? AND execute_after_at <= ?", false, now).
		Order("execute_after_at ASC, id ASC").
		Find(&terms).Error
	if err != nil {
		return nil, err
	}
	return terms, nil
}

func (t *TermRepository) ExecuteTermInterviewResults(ctx context.Context, termID uint64, now time.Time) (bool, error) {
	executed := false
	err := t.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var term model.Term
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", termID).
			First(&term).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("term 不存在")
			}
			return err
		}

		if term.IsExecuted || term.ExecuteAfterAt.After(now) {
			return nil
		}

		var interviewResults []model.InterviewResult
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("term_id = ? AND executed_at IS NULL", term.ID).
			Find(&interviewResults).Error; err != nil {
			return err
		}

		for _, result := range interviewResults {
			if err := executeInterviewResult(tx, result, now); err != nil {
				return err
			}
		}

		termTx := tx.Model(&model.Term{}).
			Where("id = ? AND is_executed = ?", term.ID, false).
			Updates(map[string]any{
				"is_executed": true,
				"executed_at": now,
			})
		if termTx.Error != nil {
			return termTx.Error
		}
		if termTx.RowsAffected == 0 {
			return errors.New("term 已被其他任务执行")
		}

		executed = true
		return nil
	})
	return executed, err
}

func executeInterviewResult(tx *gorm.DB, result model.InterviewResult, now time.Time) error {
	var user model.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id", "department_id", "role_id").
		Where("id = ?", result.UserID).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("面试结果对应用户不存在")
		}
		return err
	}

	var application model.Application
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where("id = ? AND term_id = ? AND user_id = ?", result.ApplicationID, result.TermID, result.UserID).
		First(&application).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("面试结果对应申请表不存在")
		}
		return err
	}

	appUpdate := map[string]any{
		"decision":             result.Decision,
		"result_department_id": result.ResultDepartmentID,
		"result_role_id":       result.ResultRoleID,
		"operator_id":          result.OperatorID,
		"decision_remark":      result.Remark,
	}
	if err := tx.Model(&model.Application{}).
		Where("id = ?", application.ID).
		Updates(appUpdate).Error; err != nil {
		return err
	}

	if isAdmittedDecision(result.Decision) {
		if err := tx.Model(&model.User{}).
			Where("id = ?", result.UserID).
			Updates(map[string]any{
				"department_id": result.ResultDepartmentID,
				"role_id":       result.ResultRoleID,
			}).Error; err != nil {
			return err
		}
	}

	if err := tx.Model(&model.InterviewResult{}).
		Where("id = ? AND executed_at IS NULL", result.ID).
		Updates(map[string]any{
			"old_department_id": user.DepartmentID,
			"old_role_id":       user.RoleID,
			"executed_at":       now,
		}).Error; err != nil {
		return err
	}

	return nil
}

func isAdmittedDecision(decision string) bool {
	return decision == model.DecisionAdmittedFirst ||
		decision == model.DecisionAdmittedSecond ||
		decision == model.DecisionAdmittedAdjusted
}
