package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"suseoaa/internal/model"
	"suseoaa/internal/repository"
	"suseoaa/internal/request"
	"suseoaa/pkg/logger"
	"time"

	"github.com/xuri/excelize/v2"
)

var shanghaiLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type TermService struct {
	TermRepo    *repository.TermRepository
	UserService *UserService
}

func NewTermService(termRepo *repository.TermRepository, service *UserService) *TermService {
	return &TermService{
		TermRepo:    termRepo,
		UserService: service,
	}
}

func (t *TermService) StartInterviewResultExecutor(ctx context.Context) {
	t.executeDueInterviewResultsAndLog(ctx)

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.executeDueInterviewResultsAndLog(ctx)
		}
	}
}

func (t *TermService) ExecuteDueInterviewResults(ctx context.Context) (int, error) {
	now := time.Now()
	terms, err := t.TermRepo.GetExecutableTerms(ctx, now)
	if err != nil {
		return 0, err
	}

	executed := 0
	var errs []error
	for _, term := range terms {
		termExecuted, err := t.TermRepo.ExecuteTermInterviewResults(ctx, term.ID, now)
		if err != nil {
			errs = append(errs, fmt.Errorf("term_id=%d 执行失败: %w", term.ID, err))
			continue
		}
		if termExecuted {
			executed++
		}
	}

	return executed, errors.Join(errs...)
}

func (t *TermService) executeDueInterviewResultsAndLog(ctx context.Context) {
	executed, err := t.ExecuteDueInterviewResults(ctx)
	if executed > 0 {
		logger.InfoContext(ctx, "已执行到期面试结果周期数量", "executed", executed)
	}
	if err != nil {
		logger.ErrorContext(ctx, "执行到期面试结果失败", "err", err)
	}
}

// 业务周期

func (t *TermService) CheckLevel(ctx context.Context, userID uint64) error {
	user, err := t.UserService.Repo.FindUserById(ctx, userID)
	if err != nil {
		return err
	}
	if user.IsGraduated {
		return errors.New("已毕业成员无操作权限")
	}
	level, _, err := t.UserService.Repo.GetActiveRoleLevelAndDepartment(ctx, userID)
	if err != nil {
		return err
	}
	if level < 80 {
		return errors.New("权限不够")
	}
	return nil
}

func (t *TermService) CreateTerm(ctx context.Context, term model.Term) error {
	term.Type = strings.TrimSpace(term.Type)
	if err := checkTermType(term.Type); err != nil {
		return err
	}
	err := term.CheckPeriod()
	if err != nil {
		return err
	}
	term.ExecuteAfterAt = term.QueryEndAt.Add(1 * time.Minute)
	err = t.TermRepo.CreateTerm(ctx, term)
	if err != nil {
		return err
	}
	return nil
}

func (t *TermService) UpdateTerm(ctx context.Context, term model.Term) error {
	oldTerm, err := t.TermRepo.GetTermByID(ctx, term.ID)
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
	return t.TermRepo.UpdateTerm(ctx, term)
}

func checkTermType(termType string) error {
	switch termType {
	case model.TermTypeRecruit, model.TermTypeElection:
		return nil
	default:
		return errors.New("周期类型错误")
	}
}

func (t *TermService) GetTermList(ctx context.Context, year uint64, termType string) ([]request.TermListResp, error) {
	termList, err := t.TermRepo.GetTermList(ctx, year, termType)
	if err != nil {
		return nil, err
	}
	if len(termList) == 0 {
		return []request.TermListResp{}, nil
	}

	resp := make([]request.TermListResp, 0, len(termList))
	for _, term := range termList {
		if term == nil {
			continue
		}
		resp = append(resp, request.TermListResp{
			ID:    term.ID,
			Year:  term.Year,
			Type:  term.Type,
			Title: term.Title,
			EditPeriod: request.TermPeriodResp{
				StartAt: formatTermDate(term.EditStartAt),
				EndAt:   formatTermDate(term.EditEndAt),
			},
			QueryPeriod: request.TermPeriodResp{
				StartAt: formatTermDate(term.QueryStartAt),
				EndAt:   formatTermDate(term.QueryEndAt),
			},
			IsExecuted:     term.IsExecuted,
			ExecuteAfterAt: term.ExecuteAfterAt,
			ExecutedAt:     term.ExecutedAt,
			CreatedAt:      term.CreatedAt,
			UpdatedAt:      term.UpdatedAt,
		})
	}
	return resp, nil
}

func formatTermDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format("2006-01-02")
}

func (t *TermService) DeleteTerm(ctx context.Context, userID uint64, termID uint64) error {
	if err := t.CheckLevel(ctx, userID); err != nil {
		return err
	}

	term, err := t.TermRepo.GetTermByID(ctx, termID)
	if err != nil {
		return err
	}
	if term.IsExecuted {
		return errors.New("该周期已经执行，不能删除")
	}

	return t.TermRepo.DeleteTerm(ctx, termID)
}

// 申请表

func (t *TermService) CreateApplication(ctx context.Context, application model.Application) error {
	user, err := t.UserService.Repo.FindUserById(ctx, application.UserID)
	if err != nil {
		return err
	}
	if user.IsGraduated {
		return errors.New("已毕业成员不可填报申请表")
	}
	userRole, err := t.UserService.RoleRepo.GetRoleByID(ctx, user.RoleID)
	if err != nil {
		return err
	}
	application.AvatarURI = strings.TrimSpace(application.AvatarURI)
	if err := t.checkApplicationAvatar(application.AvatarURI); err != nil {
		return err
	}

	term, err := t.TermRepo.GetTermByID(ctx, application.TermID)
	if err != nil {
		return err
	}
	if !term.IsInEditPeriod(time.Now()) {
		return errors.New("不在时间范围内")
	}
	if err := t.checkApplicationChoices(ctx, userRole, application); err != nil {
		return err
	}
	application.Type = term.Type
	return t.TermRepo.CreateApplication(ctx, application)
}

func (t *TermService) UpdateApplication(ctx context.Context, application model.Application) error {
	user, err := t.UserService.Repo.FindUserById(ctx, application.UserID)
	if err != nil {
		return err
	}
	if user.IsGraduated {
		return errors.New("已毕业成员不可修改申请表")
	}
	userRole, err := t.UserService.RoleRepo.GetRoleByID(ctx, user.RoleID)
	if err != nil {
		return err
	}

	oldApplication, err := t.TermRepo.GetLatestApplicationByUserID(ctx, application.UserID)
	if err != nil {
		return err
	}
	term, err := t.TermRepo.GetTermByID(ctx, oldApplication.TermID)
	if err != nil {
		return err
	}
	if !term.IsInEditPeriod(time.Now()) {
		return errors.New("不在时间范围内")
	}
	if err := t.checkApplicationChoices(ctx, userRole, application); err != nil {
		return err
	}

	application.AvatarURI = strings.TrimSpace(application.AvatarURI)
	if application.AvatarURI == "" {
		application.AvatarURI = oldApplication.AvatarURI
	} else if application.AvatarURI != oldApplication.AvatarURI {
		if err := t.checkApplicationAvatar(application.AvatarURI); err != nil {
			return err
		}
	}

	application.TermID = oldApplication.TermID
	application.UserID = oldApplication.UserID
	application.Type = oldApplication.Type
	if err := t.TermRepo.UpdateApplication(ctx, application); err != nil {
		return err
	}
	t.deleteOldApplicationAvatar(ctx, application.UserID, oldApplication.AvatarURI, application.AvatarURI)
	return nil
}

func (t *TermService) checkApplicationAvatar(avatar string) error {
	avatar = strings.TrimSpace(avatar)
	if avatar == "" {
		return nil
	}
	if !strings.HasPrefix(avatar, "application/") {
		return errors.New("申请表照片路径错误")
	}
	if strings.Contains(avatar, "..") {
		return errors.New("申请表照片路径不合法")
	}
	ext := strings.ToLower(filepath.Ext(avatar))
	if !allowedExt[ext] {
		return errors.New("申请表照片类型错误")
	}
	return nil
}

func (t *TermService) deleteOldApplicationAvatar(ctx context.Context, userID uint64, oldAvatar string, newAvatar string) {
	if oldAvatar == "" || oldAvatar == newAvatar {
		return
	}
	if t.UserService == nil || t.UserService.File == nil || t.UserService.File.ImgStorage == nil {
		return
	}
	if !strings.HasPrefix(oldAvatar, "application/") {
		return
	}
	if err := t.UserService.File.ImgStorage.DeleteFile(ctx, oldAvatar); err != nil {
		logger.WarnContext(ctx, "删除旧申请表照片失败", "user_id", userID, "avatar", oldAvatar, "err", err)
	}
}

func (t *TermService) GetMyApplications(ctx context.Context, userID uint64) ([]*model.Application, error) {
	applications, err := t.TermRepo.GetApplicationsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(applications) == 0 {
		return applications, nil
	}

	termMap, err := t.TermRepo.GetTermMap(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	for _, application := range applications {
		if application == nil {
			continue
		}
		term, ok := termMap[application.TermID]
		if ok {
			application.TermTitle = term.Title
		}
		// 需等待 query 周期结束（now.After(term.QueryEndAt)），候选人才能看到自己的状态；未结束前对候选人保密，显示为待定
		if !ok || !term.IsAfterQueryPeriod(now) {
			application.Decision = model.DecisionPending
			application.Result = model.OrganizationRole{}
			application.OperatorID = 0
			application.DecisionRemark = ""
		}
	}

	t.fillApplicationAvatars(ctx, applications)
	return applications, nil
}

func (t *TermService) fillApplicationTermTitles(ctx context.Context, applications []*model.Application) error {
	if len(applications) == 0 {
		return nil
	}

	termMap, err := t.TermRepo.GetTermMap(ctx)
	if err != nil {
		return err
	}

	for _, application := range applications {
		if application == nil {
			continue
		}
		if term, ok := termMap[application.TermID]; ok {
			application.TermTitle = term.Title
		}
	}
	return nil
}

func (t *TermService) fillApplicationAvatars(ctx context.Context, applications []*model.Application) {
	if t.UserService == nil || t.UserService.File == nil || t.UserService.File.ImgStorage == nil {
		return
	}
	for _, application := range applications {
		if application == nil {
			continue
		}

		application.Avatar.URI = application.AvatarURI
		if application.AvatarURI == "" {
			continue
		}

		url, err := t.UserService.File.ImgStorage.GeneratePresignedURL(ctx, application.AvatarURI)
		if err != nil {
			logger.WarnContext(ctx, "生成申请表照片链接失败", "application_id", application.ID, "avatar", application.AvatarURI, "err", err)
			continue
		}
		application.Avatar.URL = url
	}
}

func (t *TermService) checkApplicationChoices(ctx context.Context, currentUserRole *model.Role, application model.Application) error {
	if err := t.checkApplicationChoice(ctx, currentUserRole, application.FirstChoice); err != nil {
		return err
	}
	if err := t.checkApplicationChoice(ctx, currentUserRole, application.SecondChoice); err != nil {
		return err
	}
	return nil
}

func (t *TermService) checkApplicationChoice(ctx context.Context, currentUserRole *model.Role, choice model.OrganizationRole) error {
	if err := t.UserService.VerifyDepartmentPosition(ctx, choice.DepartmentID, choice.RoleID); err != nil {
		return err
	}
	if err := t.checkApplicationRole(ctx, choice.RoleID); err != nil {
		return err
	}
	if currentUserRole != nil && currentUserRole.Name == "会员" {
		choiceRole, err := t.UserService.RoleRepo.GetRoleByID(ctx, choice.RoleID)
		if err != nil {
			return err
		}
		if choiceRole != nil && choiceRole.Name == "会员" {
			return errors.New("当前已是协会会员，不可申请成为会员")
		}
	}
	return nil
}

func (t *TermService) checkApplicationRole(ctx context.Context, roleID uint64) error {
	level, err := t.UserService.RoleRepo.GetLevelByID(ctx, roleID)
	if err != nil {
		return err
	}
	if level > 90 {
		return errors.New("不能申请成为该职位")
	}
	return nil
}

func (t *TermService) GetRolesByDepartmentsID(ctx context.Context, departmentID uint64) ([]*model.Role, error) {
	var departmentType string
	var err error
	departmentType = ""
	if departmentID != 0 {
		departmentType, err = t.UserService.DepartmentRepo.GetTypeByDepartmentID(ctx, departmentID)
		if err != nil {
			return nil, err
		}
	}
	roles, err := t.UserService.RoleRepo.GetRoleByType(ctx, departmentType)
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

func (t *TermService) GetDepartmentByRoleID(ctx context.Context, roleID uint64) ([]*model.Department, error) {
	var roleType string
	var err error
	roleType = ""
	if roleID != 0 {
		roleType, err = t.UserService.RoleRepo.GetTypeByRoleID(ctx, roleID)
		if err != nil {
			return nil, err
		}
	}
	departments, err := t.UserService.DepartmentRepo.GetDepartmentByType(ctx, roleType)
	if err != nil {
		return nil, err
	}
	return departments, nil
}

func (t *TermService) resolveApplicationListScope(ctx context.Context, userID uint64, termID uint64, requestedDepartmentID uint64) (uint64, error) {
	user, err := t.UserService.Repo.FindUserById(ctx, userID)
	if err != nil {
		return 0, err
	}
	if user.IsGraduated {
		return 0, errors.New("已毕业成员无权限查看申请")
	}

	level, _, err := t.UserService.Repo.GetActiveRoleLevelAndDepartment(ctx, userID)
	if err != nil {
		return 0, err
	}

	if level >= 80 {
		return requestedDepartmentID, nil
	}

	interviewers, err := t.TermRepo.GetInterviewerListByTermID(ctx, termID)
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

func (t *TermService) GetApplicationList(ctx context.Context, userID uint64, departmentID uint64, termID uint64, hasResult *bool) ([]*model.Application, error) {
	term, err := t.TermRepo.GetTermByID(ctx, termID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	// 在招新换届时间段内（填报修改期或评审决选期）均可查看申请表
	if !(term.IsInQueryPeriod(now) || term.IsInEditPeriod(now) || (!now.Before(term.EditStartAt) && !now.After(term.QueryEndAt))) {
		return nil, errors.New("不在查询时间范围内")
	}

	finalDepartmentID, err := t.resolveApplicationListScope(ctx, userID, termID, departmentID)
	if err != nil {
		return nil, err
	}

	applications, err := t.TermRepo.GetApplicationsByTermIDAndDepartmentID(ctx, termID, finalDepartmentID, hasResult)
	if err != nil {
		return nil, err
	}
	for _, application := range applications {
		if application != nil {
			application.TermTitle = term.Title
		}
	}
	t.fillApplicationAvatars(ctx, applications)
	return applications, nil
}

func (t *TermService) DeleteApplication(ctx context.Context, applicationID uint64, id uint64) error {
	user, err := t.UserService.Repo.FindUserById(ctx, id)
	if err != nil {
		return err
	}
	if user.IsGraduated {
		return errors.New("已毕业成员无操作权限")
	}

	application, err := t.TermRepo.GetApplicationByID(ctx, applicationID)
	if err != nil {
		return err
	}
	term, err := t.TermRepo.GetTermByID(ctx, application.TermID)
	if err != nil {
		return err
	}
	ok := term.IsInEditPeriod(time.Now())
	if !ok {
		return errors.New("不在时间范围内")
	}
	level, _, err := t.UserService.Repo.GetActiveRoleLevelAndDepartment(ctx, id)
	if err != nil {
		return err
	}
	applicationLevel, _, err := t.UserService.Repo.GetRoleLevelAndDepartment(ctx, application.UserID)
	if err != nil {
		return err
	}
	if (level > applicationLevel && level >= 80) || application.UserID == id {
		err = t.TermRepo.DeleteApplication(ctx, applicationID)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("权限不够")
}

// 面试官

func (t *TermService) CreateInterviewers(ctx context.Context, id uint64, req request.CreateInterviewer) error {
	if err := t.CheckLevel(ctx, id); err != nil {
		return err
	}
	if len(req.Interviewers) == 0 {
		return errors.New("面试官列表不能为空")
	}
	if err := t.ensureInterviewerTermEditable(ctx, req.TermID); err != nil {
		return err
	}

	existing, err := t.TermRepo.GetInterviewerListByTermID(ctx, req.TermID)
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

	if len(userIDs) == 0 {
		return errors.New("全部都已经是面试官")
	}

	users, err := t.UserService.Repo.FindUsersByIDs(ctx, userIDs)
	if err != nil {
		return err
	}
	userMap := make(map[uint64]model.User, len(users))
	for _, u := range users {
		if u.IsGraduated {
			return errors.New("存在已毕业成员，不可设置为面试官")
		}
		userMap[u.ID] = u
	}

	interviewers := make([]model.Interviewer, 0, len(req.Interviewers))
	for _, item := range req.Interviewers {
		if status[item.UserID] {
			continue
		}
		u, ok := userMap[item.UserID]
		if !ok || u.DepartmentID == 0 {
			return errors.New("存在用户缺少有效部门，不能创建面试官")
		}
		status[item.UserID] = true
		interviewers = append(interviewers, model.Interviewer{
			UserID:       item.UserID,
			TermID:       req.TermID,
			Remark:       item.Remark,
			DepartmentID: u.DepartmentID,
		})
	}

	return t.TermRepo.CreateInterviewers(ctx, interviewers)
}

func (t *TermService) GetInterviewerList(ctx context.Context, userID uint64, termID uint64) ([]model.InterviewerInfo, error) {
	user, err := t.UserService.Repo.FindUserById(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.IsGraduated {
		return nil, errors.New("已毕业成员无权限查看面试官列表")
	}

	departmentID, _, err := t.UserService.Repo.GetDepartmentIDAndRoleIDByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	isAdmin := t.CheckLevel(ctx, userID) == nil
	if !isAdmin {
		all, err := t.TermRepo.GetInterviewerListByTermID(ctx, termID)
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

	interviews, err := t.TermRepo.GetInterviewerListByScope(ctx, termID, scopeDepartmentID)
	if err != nil {
		return nil, err
	}
	if len(interviews) == 0 {
		return nil, errors.New("无匹配面试官")
	}
	return t.InterviewerToInterviewerInfo(ctx, interviews)
}

func (t *TermService) UpdateInterviewer(ctx context.Context, id uint64, req request.UpdateInterviewer) error {
	if err := t.CheckLevel(ctx, id); err != nil {
		return err
	}

	interviewer, err := t.TermRepo.GetInterviewerByID(ctx, req.InterviewerID)
	if err != nil {
		return err
	}
	if err = t.ensureInterviewerTermEditable(ctx, interviewer.TermID); err != nil {
		return err
	}

	return t.TermRepo.UpdateInterviewers(ctx, model.Interviewer{
		ID:     req.InterviewerID,
		Remark: req.Remark,
	})
}

func (t *TermService) DeleteInterviewer(ctx context.Context, userID uint64, interviewerID uint64) error {
	if err := t.CheckLevel(ctx, userID); err != nil {
		return err
	}

	interviewer, err := t.TermRepo.GetInterviewerByID(ctx, interviewerID)
	if err != nil {
		return err
	}
	if err = t.ensureInterviewerTermEditable(ctx, interviewer.TermID); err != nil {
		return err
	}

	return t.TermRepo.DeleteInterviewer(ctx, interviewerID)
}

func (t *TermService) InterviewerToInterviewerInfo(ctx context.Context, interviews []model.Interviewer) ([]model.InterviewerInfo, error) {
	termMap, err := t.TermRepo.GetTermMap(ctx)
	if err != nil {
		return nil, err
	}

	ids := make([]uint64, 0, len(interviews))
	for _, item := range interviews {
		ids = append(ids, item.UserID)
	}

	usersMap, err := t.UserService.Repo.GetUserMapByUserIDs(ctx, ids)
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

func (t *TermService) ensureInterviewerTermEditable(ctx context.Context, termID uint64) error {
	term, err := t.TermRepo.GetTermByID(ctx, termID)
	if err != nil {
		return err
	}
	if term.IsExecuted {
		return errors.New("该周期已经执行，不能再修改面试官")
	}
	return nil
}

// 面试结果

func (t *TermService) CreateInterviewResult(ctx context.Context, operatorID uint64, req request.CreateInterviewResultReq) error {
	application, err := t.TermRepo.GetApplicationByID(ctx, req.ApplicationID)
	if err != nil {
		return err
	}

	if _, err := t.TermRepo.GetInterviewResultByApplicationID(ctx, req.ApplicationID); err == nil {
		return errors.New("该申请表的面试结果已存在")
	}

	term, err := t.TermRepo.GetTermByID(ctx, application.TermID)
	if err != nil {
		return err
	}

	interviewResult := model.InterviewResult{
		ApplicationID:      application.ID,
		TermID:             application.TermID,
		Type:               application.Type,
		UserID:             application.UserID,
		OperatorID:         operatorID,
		Decision:           req.Decision,
		ResultDepartmentID: req.ResultDepartmentID,
		ResultRoleID:       req.ResultRoleID,
		Remark:             req.Remark,
	}

	if err := t.CheckInterviewResult(ctx, operatorID, term, application, interviewResult); err != nil {
		return err
	}

	return t.TermRepo.CreateInterviewResult(ctx, interviewResult)
}

func (t *TermService) UpdateInterviewResult(ctx context.Context, operatorID uint64, req request.UpdateInterviewResultReq) error {
	oldResult, err := t.TermRepo.GetInterviewResultByApplicationID(ctx, req.ApplicationID)
	if err != nil {
		return err
	}

	application, err := t.TermRepo.GetApplicationByID(ctx, oldResult.ApplicationID)
	if err != nil {
		return err
	}

	term, err := t.TermRepo.GetTermByID(ctx, application.TermID)
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

	if err := t.CheckInterviewResult(ctx, operatorID, term, application, interviewResult); err != nil {
		return err
	}

	return t.TermRepo.UpdateInterviewResult(ctx, interviewResult)
}

func (t *TermService) DeleteInterviewResult(ctx context.Context, operatorID uint64, applicationID uint64) error {
	if err := t.CheckLevel(ctx, operatorID); err != nil {
		return err
	}

	oldResult, err := t.TermRepo.GetInterviewResultByApplicationID(ctx, applicationID)
	if err != nil {
		return err
	}

	application, err := t.TermRepo.GetApplicationByID(ctx, oldResult.ApplicationID)
	if err != nil {
		return err
	}

	term, err := t.TermRepo.GetTermByID(ctx, application.TermID)
	if err != nil {
		return err
	}

	if term.IsExecuted {
		return errors.New("该周期已经执行，不能再修改面试结果")
	}

	if !term.IsInQueryPeriod(time.Now()) {
		return errors.New("不在时间范围内")
	}

	return t.TermRepo.DeleteInterviewResult(ctx, oldResult.ID, oldResult.ApplicationID)
}

func (t *TermService) GetInterviewResultList(ctx context.Context, operatorID uint64, termID uint64) ([]model.InterviewResultInfo, error) {
	if err := t.CheckLevel(ctx, operatorID); err != nil {
		return nil, err
	}

	if termID != 0 {
		if _, err := t.TermRepo.GetTermByID(ctx, termID); err != nil {
			return nil, err
		}
	}

	return t.TermRepo.GetInterviewResultList(ctx, termID)
}

func (t *TermService) CheckInterviewResult(
	ctx context.Context,
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

	if term.IsExecuted {
		return errors.New("该周期已经执行，不能再修改面试结果")
	}

	if !term.IsInQueryPeriod(time.Now()) {
		return errors.New("不在时间范围内")
	}

	if err := t.CheckLevel(ctx, operatorID); err != nil {
		return err
	}

	applicant, err := t.UserService.Repo.FindUserById(ctx, application.UserID)
	if err != nil {
		return err
	}
	if applicant.IsGraduated && isAdmittedDecision(interviewResult.Decision) {
		return errors.New("该申请人已毕业，不可录取为在任职务")
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
			ctx,
			interviewResult.ResultDepartmentID,
			interviewResult.ResultRoleID,
		); err != nil {
			return err
		}
		if err := t.checkApplicationRole(ctx, interviewResult.ResultRoleID); err != nil {
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

func isAdmittedDecision(decision string) bool {
	return decision == model.DecisionAdmittedFirst ||
		decision == model.DecisionAdmittedSecond ||
		decision == model.DecisionAdmittedAdjusted
}

func (t *TermService) ExportInterviewResults(ctx context.Context, operatorID uint64, req request.ExportInterviewResultReq) ([]byte, string, error) {
	if err := t.CheckLevel(ctx, operatorID); err != nil {
		return nil, "", err
	}

	var termTitle string
	if req.TermID != 0 {
		term, err := t.TermRepo.GetTermByID(ctx, req.TermID)
		if err != nil {
			return nil, "", err
		}
		termTitle = term.Title
	}

	deptMap, err := t.UserService.DepartmentRepo.GetDepartmentMap(ctx)
	if err != nil {
		return nil, "", err
	}
	if req.DepartmentID != 0 {
		if _, ok := deptMap[req.DepartmentID]; !ok {
			return nil, "", errors.New("部门不存在")
		}
	}

	roleMap, err := t.UserService.RoleRepo.GetRoleMap(ctx)
	if err != nil {
		return nil, "", err
	}

	termMap, err := t.TermRepo.GetTermMap(ctx)
	if err != nil {
		return nil, "", err
	}

	applications, err := t.TermRepo.GetApplicationsByTermIDAndDepartmentID(ctx, req.TermID, req.DepartmentID, nil)
	if err != nil {
		return nil, "", err
	}

	operatorIDs := make([]uint64, 0)
	seen := make(map[uint64]bool)
	for _, app := range applications {
		if app != nil && app.OperatorID > 0 && !seen[app.OperatorID] {
			seen[app.OperatorID] = true
			operatorIDs = append(operatorIDs, app.OperatorID)
		}
	}

	operatorMap := make(map[uint64]model.UserInfo)
	if len(operatorIDs) > 0 {
		if users, err := t.UserService.Repo.GetUserMapByUserIDs(ctx, operatorIDs); err == nil {
			operatorMap = users
		}
	}

	mask, err := parseExportFieldMask(req.Fields, len(exportFieldDefinitions))
	if err != nil {
		return nil, "", err
	}
	if mask == 0 || (mask&request.ExportFieldAllMask) == 0 {
		return nil, "", errors.New("未选择任何有效导出字段")
	}

	type exportColumn struct {
		def         request.ExportFieldDef
		isLeftAlign bool
		getValue    func(app *model.Application, index int) string
	}

	allColumns := []exportColumn{
		{
			def: exportFieldDefinitions[request.ExportFieldBitTermTitle],
			getValue: func(app *model.Application, index int) string {
				if term, ok := termMap[app.TermID]; ok && term.Title != "" {
					return term.Title
				} else if app.TermTitle != "" {
					return app.TermTitle
				} else if termTitle != "" {
					return termTitle
				}
				return "-"
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitType],
			getValue: func(app *model.Application, index int) string {
				if app.Type != "" {
					return app.Type
				}
				return "-"
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitStudentID],
			getValue: func(app *model.Application, index int) string {
				return app.StudentID
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitName],
			getValue: func(app *model.Application, index int) string {
				return app.Name
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitGender],
			getValue: func(app *model.Application, index int) string {
				return app.Gender
			},
		},
		{
			def:         exportFieldDefinitions[request.ExportFieldBitCollege],
			isLeftAlign: true,
			getValue: func(app *model.Application, index int) string {
				return app.College
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitMajorClass],
			getValue: func(app *model.Application, index int) string {
				return app.MajorClass
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitPoliticalStatus],
			getValue: func(app *model.Application, index int) string {
				return app.PoliticalStatus
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitBirthDate],
			getValue: func(app *model.Application, index int) string {
				return app.BirthDate
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitPhone],
			getValue: func(app *model.Application, index int) string {
				return app.Phone
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitQQ],
			getValue: func(app *model.Application, index int) string {
				return app.QQ
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitFirstDepartment],
			getValue: func(app *model.Application, index int) string {
				return getDepartmentName(deptMap, app.FirstChoice.DepartmentID)
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitFirstRole],
			getValue: func(app *model.Application, index int) string {
				return getRoleName(roleMap, app.FirstChoice.RoleID)
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitSecondDepartment],
			getValue: func(app *model.Application, index int) string {
				return getDepartmentName(deptMap, app.SecondChoice.DepartmentID)
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitSecondRole],
			getValue: func(app *model.Application, index int) string {
				return getRoleName(roleMap, app.SecondChoice.RoleID)
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitDecision],
			getValue: func(app *model.Application, index int) string {
				if app.Decision != "" {
					return app.Decision
				}
				return model.DecisionPending
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitResultDepartment],
			getValue: func(app *model.Application, index int) string {
				return getDepartmentName(deptMap, app.Result.DepartmentID)
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitResultRole],
			getValue: func(app *model.Application, index int) string {
				return getRoleName(roleMap, app.Result.RoleID)
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitOperatorName],
			getValue: func(app *model.Application, index int) string {
				if app.OperatorID > 0 {
					if op, ok := operatorMap[app.OperatorID]; ok && op.Name != "" {
						return op.Name
					}
					return fmt.Sprintf("ID:%d", app.OperatorID)
				}
				return "-"
			},
		},
		{
			def:         exportFieldDefinitions[request.ExportFieldBitDecisionRemark],
			isLeftAlign: true,
			getValue: func(app *model.Application, index int) string {
				if app.DecisionRemark != "" {
					return app.DecisionRemark
				}
				return "-"
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitCreatedAt],
			getValue: func(app *model.Application, index int) string {
				if !app.CreatedAt.IsZero() {
					return app.CreatedAt.In(shanghaiLocation).Format("2006-01-02 15:04:05")
				}
				return "-"
			},
		},
		{
			def: exportFieldDefinitions[request.ExportFieldBitUpdatedAt],
			getValue: func(app *model.Application, index int) string {
				if !app.UpdatedAt.IsZero() {
					return app.UpdatedAt.In(shanghaiLocation).Format("2006-01-02 15:04:05")
				}
				return "-"
			},
		},
	}

	var activeColumns []exportColumn
	for _, col := range allColumns {
		if (mask & col.def.Mask) != 0 {
			activeColumns = append(activeColumns, col)
		}
	}

	if len(activeColumns) == 0 {
		return nil, "", errors.New("未选择任何有效导出字段")
	}

	headers := make([]string, len(activeColumns))
	for c, col := range activeColumns {
		headers[c] = col.def.Label
	}

	var rows [][]string
	for i, app := range applications {
		if app == nil {
			continue
		}
		row := make([]string, len(activeColumns))
		for c, col := range activeColumns {
			row[c] = col.getValue(app, i)
		}
		rows = append(rows, row)
	}

	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()

	sheetName := "面试结果"
	_ = f.SetSheetName("Sheet1", sheetName)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   11,
			Color:  "#FFFFFF",
			Family: "微软雅黑",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#2F5597"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "top", Color: "#D9D9D9", Style: 1},
			{Type: "bottom", Color: "#D9D9D9", Style: 1},
			{Type: "left", Color: "#D9D9D9", Style: 1},
			{Type: "right", Color: "#D9D9D9", Style: 1},
		},
	})

	cellCenterStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size:   10,
			Family: "微软雅黑",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "top", Color: "#E0E0E0", Style: 1},
			{Type: "bottom", Color: "#E0E0E0", Style: 1},
			{Type: "left", Color: "#E0E0E0", Style: 1},
			{Type: "right", Color: "#E0E0E0", Style: 1},
		},
	})

	cellLeftStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size:   10,
			Family: "微软雅黑",
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "top", Color: "#E0E0E0", Style: 1},
			{Type: "bottom", Color: "#E0E0E0", Style: 1},
			{Type: "left", Color: "#E0E0E0", Style: 1},
			{Type: "right", Color: "#E0E0E0", Style: 1},
		},
	})

	cellCenterAltStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size:   10,
			Family: "微软雅黑",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#F2F4F8"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "top", Color: "#E0E0E0", Style: 1},
			{Type: "bottom", Color: "#E0E0E0", Style: 1},
			{Type: "left", Color: "#E0E0E0", Style: 1},
			{Type: "right", Color: "#E0E0E0", Style: 1},
		},
	})

	cellLeftAltStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size:   10,
			Family: "微软雅黑",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#F2F4F8"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "top", Color: "#E0E0E0", Style: 1},
			{Type: "bottom", Color: "#E0E0E0", Style: 1},
			{Type: "left", Color: "#E0E0E0", Style: 1},
			{Type: "right", Color: "#E0E0E0", Style: 1},
		},
	})

	_ = f.SetRowHeight(sheetName, 1, 26)
	for c := range headers {
		cell, _ := excelize.CoordinatesToCellName(c+1, 1)
		_ = f.SetCellValue(sheetName, cell, headers[c])
		_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	for r, row := range rows {
		rowIdx := r + 2
		_ = f.SetRowHeight(sheetName, rowIdx, 22)
		isAlt := r%2 == 1
		for c, val := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, rowIdx)
			_ = f.SetCellValue(sheetName, cell, val)
			var targetStyle int
			if activeColumns[c].isLeftAlign {
				if isAlt {
					targetStyle = cellLeftAltStyle
				} else {
					targetStyle = cellLeftStyle
				}
			} else {
				if isAlt {
					targetStyle = cellCenterAltStyle
				} else {
					targetStyle = cellCenterStyle
				}
			}
			_ = f.SetCellStyle(sheetName, cell, cell, targetStyle)
		}
	}

	colWidths := make([]float64, len(headers))
	for i, h := range headers {
		colWidths[i] = float64(calculateDisplayWidth(h)) + 4
	}
	for _, row := range rows {
		for i, val := range row {
			w := float64(calculateDisplayWidth(val)) + 4
			if w > colWidths[i] {
				colWidths[i] = w
			}
		}
	}
	for i, w := range colWidths {
		if w < 10 {
			w = 10
		}
		if w > 40 {
			w = 40
		}
		colName, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheetName, colName, colName, w)
	}

	var filename string
	if termTitle != "" {
		filename = fmt.Sprintf("%s_面试结果.xlsx", sanitizeFilename(termTitle))
	} else {
		filename = "面试结果.xlsx"
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), filename, nil
}

var exportFieldDefinitions = []request.ExportFieldDef{
	{Index: 0, Bit: request.ExportFieldBitTermTitle, Mask: 1 << request.ExportFieldBitTermTitle, Key: "term_title", Label: "业务周期"},
	{Index: 1, Bit: request.ExportFieldBitType, Mask: 1 << request.ExportFieldBitType, Key: "type", Label: "业务类型"},
	{Index: 2, Bit: request.ExportFieldBitStudentID, Mask: 1 << request.ExportFieldBitStudentID, Key: "student_id", Label: "学号"},
	{Index: 3, Bit: request.ExportFieldBitName, Mask: 1 << request.ExportFieldBitName, Key: "name", Label: "姓名"},
	{Index: 4, Bit: request.ExportFieldBitGender, Mask: 1 << request.ExportFieldBitGender, Key: "gender", Label: "性别"},
	{Index: 5, Bit: request.ExportFieldBitCollege, Mask: 1 << request.ExportFieldBitCollege, Key: "college", Label: "学院"},
	{Index: 6, Bit: request.ExportFieldBitMajorClass, Mask: 1 << request.ExportFieldBitMajorClass, Key: "major_class", Label: "专业班级"},
	{Index: 7, Bit: request.ExportFieldBitPoliticalStatus, Mask: 1 << request.ExportFieldBitPoliticalStatus, Key: "political_status", Label: "政治面貌"},
	{Index: 8, Bit: request.ExportFieldBitBirthDate, Mask: 1 << request.ExportFieldBitBirthDate, Key: "birth_date", Label: "出生年月"},
	{Index: 9, Bit: request.ExportFieldBitPhone, Mask: 1 << request.ExportFieldBitPhone, Key: "phone", Label: "联系电话"},
	{Index: 10, Bit: request.ExportFieldBitQQ, Mask: 1 << request.ExportFieldBitQQ, Key: "qq", Label: "QQ"},
	{Index: 11, Bit: request.ExportFieldBitFirstDepartment, Mask: 1 << request.ExportFieldBitFirstDepartment, Key: "first_department", Label: "第一志愿部门"},
	{Index: 12, Bit: request.ExportFieldBitFirstRole, Mask: 1 << request.ExportFieldBitFirstRole, Key: "first_role", Label: "第一志愿职位"},
	{Index: 13, Bit: request.ExportFieldBitSecondDepartment, Mask: 1 << request.ExportFieldBitSecondDepartment, Key: "second_department", Label: "第二志愿部门"},
	{Index: 14, Bit: request.ExportFieldBitSecondRole, Mask: 1 << request.ExportFieldBitSecondRole, Key: "second_role", Label: "第二志愿职位"},
	{Index: 15, Bit: request.ExportFieldBitDecision, Mask: 1 << request.ExportFieldBitDecision, Key: "decision", Label: "面试决策"},
	{Index: 16, Bit: request.ExportFieldBitResultDepartment, Mask: 1 << request.ExportFieldBitResultDepartment, Key: "result_department", Label: "录取部门"},
	{Index: 17, Bit: request.ExportFieldBitResultRole, Mask: 1 << request.ExportFieldBitResultRole, Key: "result_role", Label: "录取职位"},
	{Index: 18, Bit: request.ExportFieldBitOperatorName, Mask: 1 << request.ExportFieldBitOperatorName, Key: "operator_name", Label: "审核人"},
	{Index: 19, Bit: request.ExportFieldBitDecisionRemark, Mask: 1 << request.ExportFieldBitDecisionRemark, Key: "decision_remark", Label: "决策备注"},
	{Index: 20, Bit: request.ExportFieldBitCreatedAt, Mask: 1 << request.ExportFieldBitCreatedAt, Key: "created_at", Label: "提交时间"},
	{Index: 21, Bit: request.ExportFieldBitUpdatedAt, Mask: 1 << request.ExportFieldBitUpdatedAt, Key: "updated_at", Label: "更新时间"},
}

func (t *TermService) GetExportFieldsDefinition() request.ExportFieldsResp {
	defs := make([]request.ExportFieldDef, len(exportFieldDefinitions))
	copy(defs, exportFieldDefinitions)
	return request.ExportFieldsResp{
		TotalFields: len(defs),
		AllBits:     strings.Repeat("1", len(defs)),
		AllMask:     request.ExportFieldAllMask,
		Fields:      defs,
	}
}

func parseExportFieldMask(raw string, numFields int) (uint64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" || strings.ToLower(raw) == "all" || strings.ToLower(raw) == "null" {
		return (uint64(1) << numFields) - 1, nil
	}

	// 1. 二进制前缀 (0b / 0B)
	if strings.HasPrefix(raw, "0b") || strings.HasPrefix(raw, "0B") {
		mask, err := strconv.ParseUint(raw[2:], 2, 64)
		if err != nil {
			return 0, fmt.Errorf("二进制字段掩码格式不合法: %s", raw)
		}
		return mask, nil
	}

	// 2. 十六进制前缀 (0x / 0X)
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		mask, err := strconv.ParseUint(raw[2:], 16, 64)
		if err != nil {
			return 0, fmt.Errorf("十六进制字段掩码格式不合法: %s", raw)
		}
		return mask, nil
	}

	// 3. 纯 0/1 状态序列字符串（如 "010101..." 或 "1111111111111111111111"）
	isBinaryStr := true
	for _, ch := range raw {
		if ch != '0' && ch != '1' {
			isBinaryStr = false
			break
		}
	}

	if isBinaryStr {
		// 从左到右，第 i 个字符对应下标为 i 的字段
		var mask uint64
		for i := 0; i < len(raw) && i < numFields; i++ {
			if raw[i] == '1' {
				mask |= uint64(1) << i
			}
		}
		if mask == 0 {
			return 0, errors.New("未选择任何有效导出字段")
		}
		return mask, nil
	}

	// 4. 兼容十进制整型掩码（如 4194303 或 32780）
	mask, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("字段掩码参数不合法: %s", raw)
	}
	return mask, nil
}

func calculateDisplayWidth(s string) int {
	width := 0
	for _, r := range s {
		if r > 127 {
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

func sanitizeFilename(name string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\r", "\n"}
	for _, char := range invalidChars {
		name = strings.ReplaceAll(name, char, "_")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "面试结果"
	}
	return name
}

func getDepartmentName(deptMap map[uint64]*model.Department, id uint64) string {
	if id == 0 {
		return "-"
	}
	if dept, ok := deptMap[id]; ok && dept != nil && dept.Name != "" {
		return dept.Name
	}
	return "-"
}

func getRoleName(roleMap map[uint64]*model.Role, id uint64) string {
	if id == 0 {
		return "-"
	}
	if role, ok := roleMap[id]; ok && role != nil && role.Name != "" {
		return role.Name
	}
	return "-"
}
