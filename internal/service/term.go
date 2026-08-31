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
	if level < 15 {
		return errors.New("用户职位太低")
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
	term.ExecuteAfterAt = term.QueryEndAt.Add(1 * time.Minute)
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
	term.ExecuteAfterAt = term.QueryEndAt.Add(1 * time.Minute)
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
	if !term.IsInEditPeriod(time.Now()) {
		return errors.New("不在时间范围内")
	}
	if err := t.checkApplicationChoices(application); err != nil {
		return err
	}
	application.TermID = oldApplication.TermID
	application.UserID = oldApplication.UserID
	application.Type = oldApplication.Type
	return t.TermRepo.UpdateApplication(application)
}

func (t *TermService) GetMyApplications(userID uint64) ([]*model.Application, error) {
	applications, err := t.TermRepo.GetApplicationsByUserID(userID)
	if err != nil {
		return nil, err
	}
	return applications, nil
}
func (t *TermService) checkApplicationChoices(application model.Application) error {
	if application.FirstChoice.DepartmentID == application.SecondChoice.DepartmentID &&
		application.FirstChoice.RoleID == application.SecondChoice.RoleID {
		return errors.New("第一志愿和第二志愿不能完全相同")
	}

	if err := t.UserService.VerifyDepartmentPosition(application.FirstChoice.DepartmentID, application.FirstChoice.RoleID); err != nil {
		return err
	}
	if err := t.UserService.VerifyDepartmentPosition(application.SecondChoice.DepartmentID, application.SecondChoice.RoleID); err != nil {
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

func (t *TermService) resolveApplicationListScope(userID uint64, termID uint64, requestedDepartmentID uint64) (uint64, error) {
	level, _, err := t.UserService.Repo.GetRoleLevelAndDepartment(userID)
	if err != nil {
		return 0, err
	}

	if level >= 80 {
		return requestedDepartmentID, nil
	}

	interviewers, err := t.TermRepo.GetInterviewerListByTermID(termID)
	if err != nil {
		return 0, err
	}

	for _, interviewer := range interviewers {
		if interviewer.UserID != userID {
			continue
		}
		if interviewer.DepartmentID == 6 {
			return 0, nil
		}
		return interviewer.DepartmentID, nil
	}

	return 0, errors.New("无权限查看该周期申请")
}

func (t *TermService) GetApplicationList(userID uint64, departmentID uint64, termID uint64) ([]*model.Application, error) {
	term, err := t.TermRepo.GetTermByID(termID)
	if err != nil {
		return nil, err
	}
	if !term.IsInQueryPeriod(time.Now()) {
		return nil, errors.New("不在查询时间范围内")
	}

	finalDepartmentID, err := t.resolveApplicationListScope(userID, termID, departmentID)
	if err != nil {
		return nil, err
	}

	return t.TermRepo.GetApplicationsByTermIDAndDepartmentID(termID, finalDepartmentID)
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

func (t *TermService) GetInterviewerList(userID uint64, termID uint64) ([]model.InterviewerInfo, error) {
	departmentID, _, err := t.UserService.Repo.GetDepartmentIDAndRoleIDByID(userID)
	if err != nil {
		return nil, err
	}

	isAdmin := t.CheckLevel(userID) == nil
	if !isAdmin {
		all, err := t.TermRepo.GetInterviewerListByTermID(termID)
		if err != nil {
			return nil, err
		}
		hasPermission := false
		for _, item := range all {
			if item.UserID == userID {
				hasPermission = true
				break
			}
		}
		if !hasPermission {
			return nil, errors.New("无权限查看面试官列表")
		}
	}

	scopeDepartmentID := uint64(0)
	if !isAdmin {
		scopeDepartmentID = departmentID
	}

	interviews, err := t.TermRepo.GetInterviewerListByScope(termID, scopeDepartmentID)
	if err != nil {
		return nil, err
	}
	if len(interviews) == 0 {
		return nil, errors.New("无匹配面试官")
	}
	return t.InterviewerToInterviewerInfo(interviews)
}

func (t *TermService) UpdateInterviewer(id uint64, interviewer request.UpdateInterviewer) error {
	err := t.CheckLevel(id)
	if err != nil {
		return err
	}
	_, err = t.TermRepo.GetInterviewerByID(interviewer.ID)
	if err != nil {
		return err
	}
	err = t.UserService.Repo.CheckUserIsHave(interviewer.UserID)
	if err != nil {
		return err
	}
	err = t.TermRepo.CheckTermIsHave(interviewer.TermID)
	if err != nil {
		return err
	}
	return t.TermRepo.UpdateInterviewers(model.Interviewer{
		UserID: interviewer.UserID,
		TermID: interviewer.TermID,
		Remark: interviewer.Remark,
		ID:     interviewer.ID,
	})
}
func (t *TermService) InterviewerToInterviewerInfo(interviews []model.Interviewer) ([]model.InterviewerInfo, error) {
	termMap, err := t.TermRepo.GetTermMap()
	if err != nil {
		return nil, err
	}

	ids := make([]uint64, 0, len(interviews))
	for _, item := range interviews {
		ids = append(ids, item.UserID)
	}

	usersMap, err := t.UserService.Repo.GetUserMapByUserIDs(ids)
	if err != nil {
		return nil, err
	}

	result := make([]model.InterviewerInfo, 0, len(interviews))
	for _, interview := range interviews {
		userInfo, ok := usersMap[interview.UserID]
		if !ok {
			continue
		}
		term, ok := termMap[interview.TermID]
		if !ok {
			continue
		}
		result = append(result, model.InterviewerInfo{
			ID:             interview.ID,
			DepartmentName: userInfo.Department,
			TermType:       term.Type,
			Year:           term.Year,
			Remark:         interview.Remark,
			Name:           userInfo.Name,
			Role:           userInfo.Role,
		})
	}
	return result, nil
}
