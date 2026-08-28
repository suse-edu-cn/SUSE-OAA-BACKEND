package service

import (
	"errors"
	"suseoaa/internal/model"
	"suseoaa/internal/repository"
	"suseoaa/internal/request"
	"time"
)

type TermService struct {
	TermRepo    repository.TermRepository
	UserService UserService
}

func NewTermService(termRepo repository.TermRepository, service UserService) TermService {
	return TermService{
		TermRepo:    termRepo,
		UserService: service,
	}
}

//-----------------------
//业务周期

func (t *TermService) CheckLevel(userID uint64) error {
	level, _, err := t.UserService.Repo.GetRoleLevelAndDepartment(userID)
	if err != nil {
		return err
	}
	if level < 80 {
		return errors.New("权力不够")
	}
	return nil
}
func (t *TermService) CreateTerm(term model.Term) error {
	err := term.CheckPeriod()
	if err != nil {
		return err
	}
	term.ExecutedAt = term.QueryEndAt.Add(1 * time.Minute)
	err = t.TermRepo.CreateTerm(term)
	if err != nil {
		return err
	}
	return nil
}
func (t *TermService) UpdateTerm(term model.Term) error {
	err := term.CheckPeriod()
	if err != nil {
		return err
	}
	term.ExecutedAt = term.QueryEndAt.Add(1 * time.Minute)
	err = t.TermRepo.UpdateTerm(term)
	if err != nil {
		return err
	}
	return nil
}
func (t *TermService) GetTermList(year uint64, termType string) ([]*model.Term, error) {
	termList, err := t.TermRepo.GetTermList(year, termType)
	if err != nil {
		return nil, err
	}
	if len(termList) == 0 {
		return nil, errors.New("无匹配数据")
	}
	return termList, nil
}

//----------------------------
//申请表

func (t *TermService) CreateApplication(application model.Application) error {
	term, err := t.TermRepo.GetTermByID(application.TermID)
	if err != nil {
		return err
	}
	ok := term.IsInEditPeriod(time.Now())
	if !ok {
		return errors.New("不在时间范围内")
	}
	err = t.CheckApplicationDepartmentPosition(application)
	if err != nil {
		return err
	}
	application.Type = term.Type
	return t.TermRepo.CreateApplication(application)
}
func (t *TermService) UpdateApplication(application model.Application) error {
	oldApplication, err := t.TermRepo.GetLatestApplicationByUserID(application.UserID)
	if err != nil {
		return err
	}
	term, err := t.TermRepo.GetTermByID(oldApplication.TermID)
	if err != nil {
		return err
	}
	ok := term.IsInEditPeriod(time.Now())
	if !ok {
		return errors.New("不在时间范围内")
	}
	err = t.CheckApplicationDepartmentPosition(application)
	if err != nil {
		return err
	}
	application.TermID = oldApplication.TermID
	application.Type = term.Type
	application.UserID = oldApplication.UserID
	return t.TermRepo.UpdateApplication(application)
}

func (t *TermService) GetMyApplications(userID uint64) ([]*model.Application, error) {
	applications, err := t.TermRepo.GetApplicationsByUserID(userID)
	if err != nil {
		return nil, err
	}
	return applications, nil
}

func (t *TermService) CheckApplicationDepartmentPosition(application model.Application) error {
	err := t.UserService.VerifyDepartmentPosition(application.FirstChoice.DepartmentID, application.FirstChoice.RoleID)
	if err != nil {
		return err
	}
	err = t.UserService.VerifyDepartmentPosition(application.SecondChoice.DepartmentID, application.SecondChoice.RoleID)
	if err != nil {
		return err
	}
	level, err := t.UserService.RoleRepo.GetLevelByID(application.FirstChoice.RoleID)
	if err != nil {
		return err
	}
	if level > 90 {
		return errors.New("不能申请成为该职位")
	}
	return nil
}
func (t *TermService) GetRolesByDepartmentsID(departmentID uint64) ([]*model.Role, error) {
	var departmentType string
	var err error
	departmentType = ""
	if departmentID != 0 {
		departmentType, err = t.UserService.DepartmentRepo.GetTypeByDepartmentID(departmentID)
		if err != nil {
			return nil, err
		}
	}
	roles, err := t.UserService.RoleRepo.GetRoleByType(departmentType)
	if err != nil {
		return nil, err
	}
	var result []*model.Role
	for _, role := range roles {
		if role.Level <= 90 {
			result = append(result, role)
		}
	}
	return result, nil
}
func (t *TermService) GetDepartmentByRoleID(roleID uint64) ([]*model.Department, error) {
	var roleType string
	var err error
	roleType = ""
	if roleID != 0 {
		roleType, err = t.UserService.RoleRepo.GetTypeByRoleID(roleID)
		if err != nil {
			return nil, err
		}
	}
	departments, err := t.UserService.DepartmentRepo.GetDepartmentByType(roleType)
	if err != nil {
		return nil, err
	}
	return departments, nil
}

//--------------------------------------
//面试官

func (t *TermService) CreateInterviewers(id uint64, interviewer request.CreateInterviewer) error {
	err := t.CheckLevel(id)
	if err != nil {
		return err
	}
	var interviewers []model.Interviewer
	var userIDs []uint64
	status := make(map[uint64]bool)
	hasInterviewer, err := t.TermRepo.GetInterviewerListByTermID(interviewer.TermID)
	if err != nil {
		return err
	}
	for _, interviewer := range hasInterviewer { //去除已在数据库内的面试官
		status[interviewer.UserID] = true
	}
	for _, user := range interviewer.Interviewers {
		if status[user.UserID] {
			continue
		}
		userIDs = append(userIDs, user.UserID)
	}
	DepartmentIDMap, err := t.UserService.Repo.GetDepartmentByUserIDs(userIDs)
	if err != nil {
		return err
	}
	for _, value := range interviewer.Interviewers {
		if status[value.UserID] { //去重
			continue
		}
		status[value.UserID] = true
		interviewers = append(interviewers, model.Interviewer{
			UserID:       value.UserID,
			TermID:       interviewer.TermID,
			Remark:       value.Remark,
			DepartmentID: DepartmentIDMap[value.UserID],
		})
	}
	if len(interviewers) == 0 {
		return errors.New("全部都已经是面试官")
	}
	return t.TermRepo.CreateInterviewers(interviewers)
}

func (t *TermService) GetInterviewerListByTermID(termID uint64) ([]model.InterviewerInfo, error) {
	interviews, err := t.TermRepo.GetInterviewerListByTermID(termID)
	if err != nil {
		return nil, err
	}
	if len(interviews) == 0 {
		return nil, errors.New("该周期暂无面试官")
	}
	var result []model.InterviewerInfo
	termMap, err := t.TermRepo.GetTermMap()
	if err != nil {
		return nil, err
	}
	var ids []uint64
	for _, value := range interviews {
		ids = append(ids, value.UserID)
	}
	usersMap, err := t.UserService.Repo.GetUserMapByUserIDs(ids)
	for _, interview := range interviews {
		result = append(result, model.InterviewerInfo{
			DepartmentName: usersMap[interview.UserID].Department,
			TermType:       termMap[interview.TermID].Type,
			Year:           termMap[interview.TermID].Year,
			Remark:         interview.Remark,
			Name:           usersMap[interview.UserID].Name,
			Role:           usersMap[interview.UserID].Role,
		})
	}
	return result, nil
}
