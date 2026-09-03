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
	oldTerm, err := t.TermRepo.GetTermByID(term.ID)
	if err != nil {
		return err
	}
	if oldTerm.IsExecuted {
		return errors.New("该周期已经执行，不能再修改")
	}

	if err := term.CheckPeriod(); err != nil {
		return err
	}
	term.ExecuteAfterAt = term.QueryEndAt.Add(1 * time.Minute)
	return t.TermRepo.UpdateTerm(term)
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

func (t *TermService) DeleteTerm(userID uint64, termID uint64) error {
	if err := t.CheckLevel(userID); err != nil {
		return err
	}

	term, err := t.TermRepo.GetTermByID(termID)
	if err != nil {
		return err
	}
	if term.IsExecuted {
		return errors.New("该周期已经执行，不能删除")
	}

	return t.TermRepo.DeleteTerm(termID)
}

//----------------------------
//申请表

func (t *TermService) CreateApplication(application model.Application) error {
	term, err := t.TermRepo.GetTermByID(application.TermID)
	if err != nil {
		return err
	}
	if !term.IsInEditPeriod(time.Now()) {
		return errors.New("不在时间范围内")
	}
	if err := t.checkApplicationChoices(application); err != nil {
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

func (t *TermService) DeleteApplication(applicationID uint64, id uint64) error {
	application, err := t.TermRepo.GetApplicationByID(applicationID)
	if err != nil {
		return err
	}
	term, err := t.TermRepo.GetTermByID(application.TermID)
	if err != nil {
		return err
	}
	ok := term.IsInEditPeriod(time.Now())
	if !ok {
		return errors.New("不在时间范围内")
	}
	level, _, err := t.UserService.Repo.GetRoleLevelAndDepartment(id)
	if err != nil {
		return err
	}
	applicationLevel, _, err := t.UserService.Repo.GetRoleLevelAndDepartment(application.UserID)
	if err != nil {
		return err
	}
	if (level > applicationLevel && level >= 80) || application.UserID == id {
		err = t.TermRepo.DeleteApplication(applicationID)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("权限不够")

}

//--------------------------------------
//面试官

func (t *TermService) CreateInterviewers(id uint64, req request.CreateInterviewer) error {
	if err := t.CheckLevel(id); err != nil {
		return err
	}
	if len(req.Interviewers) == 0 {
		return errors.New("面试官列表不能为空")
	}
	if err := t.ensureInterviewerTermEditable(req.TermID); err != nil {
		return err
	}

	existing, err := t.TermRepo.GetInterviewerListByTermID(req.TermID)
	if err != nil {
		return err
	}
	status := make(map[uint64]bool)
	for _, item := range existing {
		status[item.UserID] = true
	}

	userIDs := make([]uint64, 0, len(req.Interviewers))
	for _, item := range req.Interviewers {
		if status[item.UserID] {
			continue
		}
		userIDs = append(userIDs, item.UserID)
	}

	departmentIDMap, err := t.UserService.Repo.GetDepartmentByUserIDs(userIDs)
	if err != nil {
		return err
	}

	interviewers := make([]model.Interviewer, 0, len(req.Interviewers))
	for _, item := range req.Interviewers {
		if status[item.UserID] {
			continue
		}
		departmentID, ok := departmentIDMap[item.UserID]
		if !ok || departmentID == 0 {
			return errors.New("存在用户缺少有效部门，不能创建面试官")
		}
		status[item.UserID] = true
		interviewers = append(interviewers, model.Interviewer{
			UserID:       item.UserID,
			TermID:       req.TermID,
			Remark:       item.Remark,
			DepartmentID: departmentID,
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

func (t *TermService) UpdateInterviewer(id uint64, req request.UpdateInterviewer) error {
	if err := t.CheckLevel(id); err != nil {
		return err
	}

	interviewer, err := t.TermRepo.GetInterviewerByID(req.InterviewerID)
	if err != nil {
		return err
	}
	if err = t.ensureInterviewerTermEditable(interviewer.TermID); err != nil {
		return err
	}

	return t.TermRepo.UpdateInterviewers(model.Interviewer{
		ID:     req.InterviewerID,
		Remark: req.Remark,
	})
}

func (t *TermService) DeleteInterviewer(userID uint64, interviewerID uint64) error {
	if err := t.CheckLevel(userID); err != nil {
		return err
	}

	interviewer, err := t.TermRepo.GetInterviewerByID(interviewerID)
	if err != nil {
		return err
	}
	if err = t.ensureInterviewerTermEditable(interviewer.TermID); err != nil {
		return err
	}

	return t.TermRepo.DeleteInterviewer(interviewerID)
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
func (t *TermService) ensureInterviewerTermEditable(termID uint64) error {
	term, err := t.TermRepo.GetTermByID(termID)
	if err != nil {
		return err
	}
	if term.IsExecuted {
		return errors.New("该周期已经执行，不能再修改面试官")
	}
	return nil
}

//面试结果

func (t *TermService) CreateInterviewResult(operatorID uint64, req request.CreateInterviewResultReq) error {
	application, err := t.TermRepo.GetApplicationByID(req.ApplicationID)
	if err != nil {
		return err
	}

	term, err := t.TermRepo.GetTermByID(application.TermID)
	if err != nil {
		return err
	}

	interviewResult := model.InterviewResult{
		ApplicationID:      application.ID,
		TermID:             application.TermID,
		UserID:             application.UserID,
		OperatorID:         operatorID,
		Decision:           req.Decision,
		ResultDepartmentID: req.ResultDepartmentID,
		ResultRoleID:       req.ResultRoleID,
		Remark:             req.Remark,
	}

	if err := t.CheckInterviewResult(operatorID, term, application, interviewResult); err != nil {
		return err
	}

	return t.TermRepo.CreateInterviewResult(interviewResult)
}

func (t *TermService) UpdateInterviewResult(operatorID uint64, req request.UpdateInterviewResultReq) error {
	oldResult, err := t.TermRepo.GetInterviewResultByID(req.InterviewResultID)
	if err != nil {
		return err
	}

	application, err := t.TermRepo.GetApplicationByID(oldResult.ApplicationID)
	if err != nil {
		return err
	}

	term, err := t.TermRepo.GetTermByID(application.TermID)
	if err != nil {
		return err
	}

	// 保留结果记录的业务关联字段，只更新允许修改的结果内容。
	interviewResult := oldResult
	interviewResult.OperatorID = operatorID
	interviewResult.Decision = req.Decision
	interviewResult.ResultDepartmentID = req.ResultDepartmentID
	interviewResult.ResultRoleID = req.ResultRoleID
	interviewResult.Remark = req.Remark

	if err := t.CheckInterviewResult(operatorID, term, application, interviewResult); err != nil {
		return err
	}

	return t.TermRepo.UpdateInterviewResult(interviewResult)
}

func (t *TermService) CheckInterviewResult(
	operatorID uint64,
	term model.Term,
	application *model.Application,
	interviewResult model.InterviewResult,
) error {
	if application == nil {
		return errors.New("申请表不存在")
	}

	if interviewResult.ApplicationID != application.ID {
		return errors.New("面试结果和申请表不匹配")
	}
	if interviewResult.TermID != term.ID || application.TermID != term.ID {
		return errors.New("面试结果和周期不匹配")
	}
	if interviewResult.UserID != application.UserID {
		return errors.New("面试结果和用户不匹配")
	}

	if !term.IsInQueryPeriod(time.Now()) {
		return errors.New("不在时间范围内")
	}

	if err := t.CheckLevel(operatorID); err != nil {
		return err
	}

	switch interviewResult.Decision {
	case model.DecisionAdmittedFirst:
		if interviewResult.ResultDepartmentID != application.FirstChoice.DepartmentID ||
			interviewResult.ResultRoleID != application.FirstChoice.RoleID {
			return errors.New("传入的志愿和实际第一志愿不符")
		}

	case model.DecisionAdmittedSecond:
		if interviewResult.ResultDepartmentID != application.SecondChoice.DepartmentID ||
			interviewResult.ResultRoleID != application.SecondChoice.RoleID {
			return errors.New("传入的志愿和实际第二志愿不符")
		}

	case model.DecisionAdmittedAdjusted:
		if application.AllowAdjust != nil && !*application.AllowAdjust {
			return errors.New("该用户不支持调剂")
		}
		if err := t.UserService.VerifyDepartmentPosition(
			interviewResult.ResultDepartmentID,
			interviewResult.ResultRoleID,
		); err != nil {
			return err
		}

	case model.DecisionRejected:
		if interviewResult.ResultDepartmentID != 0 || interviewResult.ResultRoleID != 0 {
			return errors.New("未通过时最终部门和职位必须为0")
		}

	default:
		return errors.New("传入的决定不符规范")
	}

	return nil
}

func (t *TermService) GetInterviewResultDecision() []string {
	return []string{
		model.DecisionAdmittedFirst,
		model.DecisionAdmittedSecond,
		model.DecisionRejected,
		model.DecisionAdmittedAdjusted,
	}
}
