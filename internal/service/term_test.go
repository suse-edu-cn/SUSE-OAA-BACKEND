package service

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"suseoaa/internal/model"
	"suseoaa/internal/repository"
	"suseoaa/internal/request"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestTermService(t *testing.T) (*TermService, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite in-memory db: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.Department{},
		&model.Role{},
		&model.Term{},
		&model.Application{},
		&model.Interviewer{},
		&model.InterviewResult{},
	); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	t.Cleanup(func() {
		_ = rdb.Close()
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})

	userRepo := repository.NewUserRepository(db, rdb)
	deptRepo := repository.NewDepartmentRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	termRepo := repository.NewTermRepository(db)

	userSvc := NewUserService(userRepo, roleRepo, deptRepo, nil, nil)
	termSvc := NewTermService(termRepo, userSvc)
	return termSvc, db
}

func TestTermService_GraduatedUser_Restrictions(t *testing.T) {
	svc, db := setupTestTermService(t)
	ctx := context.Background()

	// 准备基础数据
	dept := model.Department{ID: 1, Name: "技术部", Type: "部门", IsActive: true}
	db.Create(&dept)

	roleMember := model.Role{ID: 1, Name: "会员", Level: 10, Type: "部门", IsActive: true}
	roleMinister := model.Role{ID: 2, Name: "部长", Level: 70, Type: "部门", IsActive: true}
	roleVP := model.Role{ID: 3, Name: "副会长", Level: 80, Type: "理事会", IsActive: true}
	db.Create(&roleMember)
	db.Create(&roleMinister)
	db.Create(&roleVP)

	// 1. 在校高权限操作人（副会长）
	operator := model.User{
		ID:           10,
		StudentID:    "20220001",
		Username:     "operator",
		Grade:        2022,
		IsGraduated:  false,
		DepartmentID: 1,
		RoleID:       3,
	}
	db.Create(&operator)

	// 2. 已毕业高权限操作人（虽然 Level=80，但已毕业）
	graduatedOperator := model.User{
		ID:           11,
		StudentID:    "20200001",
		Username:     "grad_operator",
		Grade:        2020,
		IsGraduated:  true,
		DepartmentID: 1,
		RoleID:       3,
	}
	db.Create(&graduatedOperator)

	// 3. 普通已毕业成员
	graduatedUser := model.User{
		ID:           20,
		StudentID:    "20200002",
		Username:     "grad_user",
		Grade:        2020,
		IsGraduated:  true,
		DepartmentID: 1,
		RoleID:       1,
	}
	db.Create(&graduatedUser)

	// 4. 普通在校成员
	activeUser := model.User{
		ID:           21,
		StudentID:    "20240001",
		Username:     "active_user",
		Grade:        2024,
		IsGraduated:  false,
		DepartmentID: 1,
		RoleID:       1,
	}
	db.Create(&activeUser)

	// 5. 创建一个处于编辑期内的周期
	now := time.Now()
	term := model.Term{
		ID:             1,
		Title:          "2026秋季招新",
		Type:           model.TermTypeRecruit,
		Year:           2026,
		EditStartAt:    now.Add(-1 * time.Hour),
		EditEndAt:      now.Add(24 * time.Hour),
		QueryStartAt:   now.Add(25 * time.Hour),
		QueryEndAt:     now.Add(48 * time.Hour),
		ExecuteAfterAt: now.Add(49 * time.Hour),
	}
	db.Create(&term)

	t.Run("已毕业成员调用 CheckLevel 应被拒绝", func(t *testing.T) {
		err := svc.CheckLevel(ctx, graduatedOperator.ID)
		if err == nil || !strings.Contains(err.Error(), "已毕业成员无操作权限") {
			t.Fatalf("expected 已毕业成员无操作权限, got: %v", err)
		}

		// 正常在校副会长应通过
		if err := svc.CheckLevel(ctx, operator.ID); err != nil {
			t.Fatalf("active operator should pass CheckLevel, got: %v", err)
		}
	})

	t.Run("已毕业成员填报申请表应被拦截", func(t *testing.T) {
		app := model.Application{
			TermID: term.ID,
			UserID: graduatedUser.ID,
			FirstChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       2,
			},
			SecondChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       2,
			},
		}
		err := svc.CreateApplication(ctx, app)
		if err == nil || !strings.Contains(err.Error(), "已毕业成员不可填报申请表") {
			t.Fatalf("expected 已毕业成员不可填报申请表, got: %v", err)
		}
	})

	t.Run("已毕业成员修改申请表应被拦截", func(t *testing.T) {
		app := model.Application{
			TermID: term.ID,
			UserID: graduatedUser.ID,
			FirstChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       2,
			},
			SecondChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       2,
			},
		}
		err := svc.UpdateApplication(ctx, app)
		if err == nil || !strings.Contains(err.Error(), "已毕业成员不可修改申请表") {
			t.Fatalf("expected 已毕业成员不可修改申请表, got: %v", err)
		}
	})

	t.Run("协会会员第一志愿申请会员应被拦截", func(t *testing.T) {
		app := model.Application{
			TermID: term.ID,
			UserID: activeUser.ID,
			FirstChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       1,
			},
			SecondChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       2,
			},
		}
		err := svc.CreateApplication(ctx, app)
		if err == nil || !strings.Contains(err.Error(), "当前已是协会会员，不可申请成为会员") {
			t.Fatalf("expected '当前已是协会会员，不可申请成为会员', got: %v", err)
		}
	})

	t.Run("协会会员第二志愿申请会员应被拦截", func(t *testing.T) {
		app := model.Application{
			TermID: term.ID,
			UserID: activeUser.ID,
			FirstChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       2,
			},
			SecondChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       1,
			},
		}
		err := svc.CreateApplication(ctx, app)
		if err == nil || !strings.Contains(err.Error(), "当前已是协会会员，不可申请成为会员") {
			t.Fatalf("expected '当前已是协会会员，不可申请成为会员', got: %v", err)
		}
	})

	t.Run("现任骨干申请会员以卸任应允许通过", func(t *testing.T) {
		app := model.Application{
			TermID: term.ID,
			UserID: operator.ID,
			FirstChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       1,
			},
			SecondChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       2,
			},
		}
		err := svc.CreateApplication(ctx, app)
		if err != nil {
			t.Fatalf("officer applying for member should succeed, got: %v", err)
		}
		// 清理申请记录避免影响后续子测试
		_ = db.Where("user_id = ? AND term_id = ?", operator.ID, term.ID).Delete(&model.Application{}).Error
	})

	t.Run("设置面试官时如果候选人包含已毕业成员应被拦截", func(t *testing.T) {
		req := request.CreateInterviewer{
			TermID: term.ID,
			Interviewers: []request.Interviewer{
				{UserID: activeUser.ID, Remark: "在校干事"},
				{UserID: graduatedUser.ID, Remark: "已毕业"},
			},
		}
		err := svc.CreateInterviewers(ctx, operator.ID, req)
		if err == nil || !strings.Contains(err.Error(), "存在已毕业成员，不可设置为面试官") {
			t.Fatalf("expected 存在已毕业成员，不可设置为面试官, got: %v", err)
		}
	})

	t.Run("已毕业操作人设置面试官应被拦截", func(t *testing.T) {
		req := request.CreateInterviewer{
			TermID: term.ID,
			Interviewers: []request.Interviewer{
				{UserID: activeUser.ID, Remark: "在校干事"},
			},
		}
		err := svc.CreateInterviewers(ctx, graduatedOperator.ID, req)
		if err == nil || !strings.Contains(err.Error(), "已毕业成员无操作权限") {
			t.Fatalf("expected 已毕业成员无操作权限, got: %v", err)
		}
	})

	t.Run("正常在校成员设置为面试官成功", func(t *testing.T) {
		req := request.CreateInterviewer{
			TermID: term.ID,
			Interviewers: []request.Interviewer{
				{UserID: activeUser.ID, Remark: "在校干事"},
			},
		}
		err := svc.CreateInterviewers(ctx, operator.ID, req)
		if err != nil {
			t.Fatalf("unexpected error creating active interviewer: %v", err)
		}
	})

	t.Run("面试结果录用已毕业申请人应被拦截，拒绝状态应允许", func(t *testing.T) {
		app := model.Application{
			ID:     100,
			TermID: term.ID,
			UserID: graduatedUser.ID,
			FirstChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       2,
			},
			SecondChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       2,
			},
		}
		db.Create(&app)

		queryTerm := term
		queryTerm.QueryStartAt = now.Add(-1 * time.Hour)
		queryTerm.QueryEndAt = now.Add(24 * time.Hour)

		admitResult := model.InterviewResult{
			ApplicationID:      app.ID,
			TermID:             term.ID,
			UserID:             graduatedUser.ID,
			Decision:           model.DecisionAdmittedFirst,
			ResultDepartmentID: 1,
			ResultRoleID:       2,
		}
		err := svc.CheckInterviewResult(ctx, operator.ID, queryTerm, &app, admitResult)
		if err == nil || !strings.Contains(err.Error(), "不可录取为在任职务") {
			t.Fatalf("expected 不可录取为在任职务, got: %v", err)
		}

		rejectResult := model.InterviewResult{
			ApplicationID:      app.ID,
			TermID:             term.ID,
			UserID:             graduatedUser.ID,
			Decision:           model.DecisionRejected,
			ResultDepartmentID: 0,
			ResultRoleID:       0,
		}
		err = svc.CheckInterviewResult(ctx, operator.ID, queryTerm, &app, rejectResult)
		if err != nil {
			t.Fatalf("reject decision for graduated user should be allowed, got: %v", err)
		}
	})

	t.Run("申请表照片路径规范化与校验", func(t *testing.T) {
		// 1. 严格区分 scene：不支持 avatar/ 或其他非法前缀，必须为 application/
		if err := svc.checkApplicationAvatar("avatar/user.png"); err == nil || !strings.Contains(err.Error(), "申请表照片路径错误") {
			t.Fatalf("expected 申请表照片路径错误 for avatar/, got: %v", err)
		}
		if err := svc.checkApplicationAvatar("illegal/path.png"); err == nil || !strings.Contains(err.Error(), "申请表照片路径错误") {
			t.Fatalf("expected 申请表照片路径错误, got: %v", err)
		}

		// 2. 路径穿越校验与文件类型校验
		if err := svc.checkApplicationAvatar("application/../hack.png"); err == nil || !strings.Contains(err.Error(), "申请表照片路径不合法") {
			t.Fatalf("expected 申请表照片路径不合法, got: %v", err)
		}
		if err := svc.checkApplicationAvatar("application/hack.exe"); err == nil || !strings.Contains(err.Error(), "申请表照片类型错误") {
			t.Fatalf("expected 申请表照片类型错误, got: %v", err)
		}

		// 3. 正常路径与空路径允许通过
		if err := svc.checkApplicationAvatar("application/pic.png"); err != nil {
			t.Fatalf("valid avatar should be allowed, got: %v", err)
		}
		if err := svc.checkApplicationAvatar(""); err != nil {
			t.Fatalf("empty avatar should be allowed, got: %v", err)
		}
	})

	t.Run("修改申请表时未传新照片应保留旧照片不被误删", func(t *testing.T) {
		origApp := model.Application{
			ID:        200,
			TermID:    term.ID,
			UserID:    activeUser.ID,
			AvatarURI: "application/old_avatar.png",
			FirstChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       2,
			},
			SecondChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       2,
			},
		}
		db.Create(&origApp)

		// 提交修改，但 avatar 为空（例如只改了联系方式）
		updateReq := model.Application{
			UserID:    activeUser.ID,
			AvatarURI: "", // 未传新头像
			FirstChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       2,
			},
			SecondChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       2,
			},
		}
		if err := svc.UpdateApplication(ctx, updateReq); err != nil {
			t.Fatalf("unexpected error updating application: %v", err)
		}

		// 验证数据库中原本的照片依然完整保留，未被清空为 ""
		var savedApp model.Application
		if err := db.First(&savedApp, origApp.ID).Error; err != nil {
			t.Fatalf("failed to query saved app: %v", err)
		}
		if savedApp.AvatarURI != "application/old_avatar.png" {
			t.Fatalf("expected avatar to be preserved as 'application/old_avatar.png', got %q", savedApp.AvatarURI)
		}
	})

	t.Run("面试结果增改删同步更新申请表decision状态", func(t *testing.T) {
		candidateUser := model.User{
			ID:           30,
			StudentID:    "20230005",
			Username:     "candidate_user",
			Grade:        2023,
			IsGraduated:  false,
			DepartmentID: 1,
			RoleID:       1,
		}
		db.Create(&candidateUser)

		app := model.Application{
			ID:        300,
			TermID:    term.ID,
			UserID:    candidateUser.ID,
			Decision:  model.DecisionPending,
			FirstChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       2,
			},
			SecondChoice: model.OrganizationRole{
				DepartmentID: 1,
				RoleID:       1,
			},
		}
		db.Create(&app)

		// 确保处于 query period
		db.Model(&model.Term{}).Where("id = ?", term.ID).Updates(map[string]any{
			"query_start_at": now.Add(-1 * time.Hour),
			"query_end_at":   now.Add(24 * time.Hour),
		})

		// 1. 创建面试结果 -> 录取第一志愿
		createReq := request.CreateInterviewResultReq{
			ApplicationID:      app.ID,
			Decision:           model.DecisionAdmittedFirst,
			ResultDepartmentID: 1,
			ResultRoleID:       2,
			Remark:             "表现优异",
		}
		if err := svc.CreateInterviewResult(ctx, operator.ID, createReq); err != nil {
			t.Fatalf("CreateInterviewResult failed: %v", err)
		}

		var checkApp model.Application
		if err := db.First(&checkApp, app.ID).Error; err != nil {
			t.Fatalf("failed to query app: %v", err)
		}
		if checkApp.Decision != model.DecisionAdmittedFirst {
			t.Fatalf("expected decision %q, got %q", model.DecisionAdmittedFirst, checkApp.Decision)
		}
		if checkApp.Result.DepartmentID != 1 || checkApp.Result.RoleID != 2 {
			t.Fatalf("expected result department 1 role 2, got dept %d role %d", checkApp.Result.DepartmentID, checkApp.Result.RoleID)
		}

		// 面试官视角：查看申请表列表，能够看到录用状态改变
		interviewerApps, err := svc.GetApplicationList(ctx, operator.ID, 0, term.ID, nil)
		if err != nil {
			t.Fatalf("GetApplicationList failed: %v", err)
		}
		foundAppInList := false
		for _, ia := range interviewerApps {
			if ia.ID == app.ID {
				foundAppInList = true
				if ia.Decision != model.DecisionAdmittedFirst {
					t.Fatalf("interviewer application list should see %q, got %q", model.DecisionAdmittedFirst, ia.Decision)
				}
				if ia.Result.DepartmentID != 1 || ia.Result.RoleID != 2 {
					t.Fatalf("interviewer application list should see result dept 1 role 2, got %v", ia.Result)
				}
			}
		}
		if !foundAppInList {
			t.Fatalf("expected to find app in interviewer application list")
		}

		// 筛选 has_result 测试
		trueVal := true
		falseVal := false
		filledApps, err := svc.GetApplicationList(ctx, operator.ID, 0, term.ID, &trueVal)
		if err != nil {
			t.Fatalf("GetApplicationList with has_result=true failed: %v", err)
		}
		for _, fa := range filledApps {
			if fa.Decision == model.DecisionPending {
				t.Fatalf("expected filled apps to not be pending, got id=%d", fa.ID)
			}
		}

		unfilledApps, err := svc.GetApplicationList(ctx, operator.ID, 0, term.ID, &falseVal)
		if err != nil {
			t.Fatalf("GetApplicationList with has_result=false failed: %v", err)
		}
		for _, ufa := range unfilledApps {
			if ufa.Decision != model.DecisionPending {
				t.Fatalf("expected unfilled apps to be pending, got id=%d, decision=%s", ufa.ID, ufa.Decision)
			}
		}

		// 个人视角：query 周期未结束时，查看本人的申请表不可以看到状态改变（始终保持为待定，拟录取结果与备注脱敏清空）
		myApps, err := svc.GetMyApplications(ctx, candidateUser.ID)
		if err != nil {
			t.Fatalf("GetMyApplications failed: %v", err)
		}
		if len(myApps) == 0 {
			t.Fatalf("expected candidate application, got 0")
		}
		if myApps[0].Decision != model.DecisionPending {
			t.Fatalf("candidate personal application view must remain '待定' before query end, got %q", myApps[0].Decision)
		}
		if myApps[0].Result.DepartmentID != 0 || myApps[0].Result.RoleID != 0 {
			t.Fatalf("candidate personal application result must be masked to 0, got %v", myApps[0].Result)
		}
		if myApps[0].OperatorID != 0 || myApps[0].DecisionRemark != "" {
			t.Fatalf("candidate personal application operator and remark must be masked, got op=%d, remark=%q", myApps[0].OperatorID, myApps[0].DecisionRemark)
		}

		// 个人视角：query 周期结束后，候选人可以查看到自己的真实录用状态
		db.Model(&model.Term{}).Where("id = ?", term.ID).Updates(map[string]any{
			"query_end_at": time.Now().Add(-1 * time.Minute),
		})
		myAppsAfterQuery, err := svc.GetMyApplications(ctx, candidateUser.ID)
		if err != nil {
			t.Fatalf("GetMyApplications after query end failed: %v", err)
		}
		if myAppsAfterQuery[0].Decision != model.DecisionAdmittedFirst {
			t.Fatalf("candidate personal application should see %q after query period ends, got %q", model.DecisionAdmittedFirst, myAppsAfterQuery[0].Decision)
		}
		if myAppsAfterQuery[0].Result.DepartmentID != 1 || myAppsAfterQuery[0].Result.RoleID != 2 {
			t.Fatalf("candidate personal application should see result dept 1 role 2, got %v", myAppsAfterQuery[0].Result)
		}

		// 还原 query 周期供后续测试
		db.Model(&model.Term{}).Where("id = ?", term.ID).Updates(map[string]any{
			"query_end_at": now.Add(24 * time.Hour),
		})

		// 2. 更新面试结果 -> 未通过
		updateReq := request.UpdateInterviewResultReq{
			ApplicationID:      app.ID,
			Decision:           model.DecisionRejected,
			ResultDepartmentID: 0,
			ResultRoleID:       0,
			Remark:             "复试未通过",
		}
		if err := svc.UpdateInterviewResult(ctx, operator.ID, updateReq); err != nil {
			t.Fatalf("UpdateInterviewResult failed: %v", err)
		}

		if err := db.First(&checkApp, app.ID).Error; err != nil {
			t.Fatalf("failed to query app: %v", err)
		}
		if checkApp.Decision != model.DecisionRejected {
			t.Fatalf("expected decision %q, got %q", model.DecisionRejected, checkApp.Decision)
		}
		if checkApp.Result.DepartmentID != 0 || checkApp.Result.RoleID != 0 {
			t.Fatalf("expected result department 0 role 0, got dept %d role %d", checkApp.Result.DepartmentID, checkApp.Result.RoleID)
		}

		// 3. 删除面试结果 -> 重置回 待定
		if err := svc.DeleteInterviewResult(ctx, operator.ID, app.ID); err != nil {
			t.Fatalf("DeleteInterviewResult failed: %v", err)
		}

		if err := db.First(&checkApp, app.ID).Error; err != nil {
			t.Fatalf("failed to query app: %v", err)
		}
		if checkApp.Decision != model.DecisionPending {
			t.Fatalf("expected decision %q after delete, got %q", model.DecisionPending, checkApp.Decision)
		}
		if checkApp.OperatorID != 0 {
			t.Fatalf("expected operator_id to be reset to 0 after delete, got %d", checkApp.OperatorID)
		}

		// 验证删除面试结果后 has_result 筛选恢复：has_result=false 包含该申请，has_result=true 不包含
		appsAfterDeleteFilled, err := svc.GetApplicationList(ctx, operator.ID, 0, term.ID, &trueVal)
		if err != nil {
			t.Fatalf("GetApplicationList filled after delete failed: %v", err)
		}
		for _, fa := range appsAfterDeleteFilled {
			if fa.ID == app.ID {
				t.Fatalf("deleted interview result should not appear in has_result=true list")
			}
		}
		appsAfterDeleteUnfilled, err := svc.GetApplicationList(ctx, operator.ID, 0, term.ID, &falseVal)
		if err != nil {
			t.Fatalf("GetApplicationList unfilled after delete failed: %v", err)
		}
		foundAfterDelete := false
		for _, ufa := range appsAfterDeleteUnfilled {
			if ufa.ID == app.ID {
				foundAfterDelete = true
				break
			}
		}
		if !foundAfterDelete {
			t.Fatalf("deleted interview result should appear in has_result=false list")
		}

		var resCount int64
		db.Model(&model.InterviewResult{}).Where("application_id = ?", app.ID).Count(&resCount)
		if resCount != 0 {
			t.Fatalf("expected interview result to be deleted, count = %d", resCount)
		}

		// 4. 重新录入面试结果 -> 再次成功且 decision 再次同步
		if err := svc.CreateInterviewResult(ctx, operator.ID, createReq); err != nil {
			t.Fatalf("Re-creating InterviewResult after delete failed: %v", err)
		}
		if err := db.First(&checkApp, app.ID).Error; err != nil {
			t.Fatalf("failed to query app: %v", err)
		}
		if checkApp.Decision != model.DecisionAdmittedFirst {
			t.Fatalf("expected decision %q after recreate, got %q", model.DecisionAdmittedFirst, checkApp.Decision)
		}

		// 5. 重复创建应被拦截
		if err := svc.CreateInterviewResult(ctx, operator.ID, createReq); err == nil || !strings.Contains(err.Error(), "已存在") {
			t.Fatalf("expected duplicate error, got: %v", err)
		}

		// 6. 权限不足操作人删除应被拦截
		if err := svc.DeleteInterviewResult(ctx, activeUser.ID, app.ID); err == nil || !strings.Contains(err.Error(), "权限不够") {
			t.Fatalf("expected permission error for low level operator, got: %v", err)
		}
		if err := svc.DeleteInterviewResult(ctx, graduatedOperator.ID, app.ID); err == nil || !strings.Contains(err.Error(), "已毕业成员无操作权限") {
			t.Fatalf("expected graduated operator error, got: %v", err)
		}

		// 7. 删除不存在的申请表面试结果应报错
		if err := svc.DeleteInterviewResult(ctx, operator.ID, 999999); err == nil || !strings.Contains(err.Error(), "不存在") {
			t.Fatalf("expected non-existent error, got: %v", err)
		}
	})

	t.Run("招新换届填报修改期内可以查看申请表但不能修改面试结果", func(t *testing.T) {
		// 处于填报修改期 (edit_period)
		editTerm := model.Term{
			ID:             20,
			Title:          "2026春招-填报期",
			Type:           model.TermTypeRecruit,
			Year:           2027,
			EditStartAt:    now.Add(-2 * time.Hour),
			EditEndAt:      now.Add(2 * time.Hour),
			QueryStartAt:   now.Add(3 * time.Hour),
			QueryEndAt:     now.Add(24 * time.Hour),
			ExecuteAfterAt: now.Add(25 * time.Hour),
		}
		db.Create(&editTerm)

		editCandidate := model.User{
			ID:           40,
			StudentID:    "20230040",
			Username:     "edit_cand",
			Grade:        2023,
			IsGraduated:  false,
			DepartmentID: 1,
			RoleID:       1,
		}
		db.Create(&editCandidate)

		editApp := model.Application{
			ID:          400,
			TermID:      editTerm.ID,
			UserID:      editCandidate.ID,
			Decision:    model.DecisionPending,
			FirstChoice: model.OrganizationRole{DepartmentID: 1, RoleID: 2},
		}
		db.Create(&editApp)

		// 1. 修改期内管理员/面试官允许调用 GetApplicationList 查看申请表
		apps, err := svc.GetApplicationList(ctx, operator.ID, 0, editTerm.ID, nil)
		if err != nil {
			t.Fatalf("expected GetApplicationList to succeed during edit period, got: %v", err)
		}
		if len(apps) == 0 {
			t.Fatalf("expected applications returned, got 0")
		}

		// 2. 修改期内不能录入面试结果 (只能查看不能修改)
		createReq := request.CreateInterviewResultReq{
			ApplicationID:      editApp.ID,
			Decision:           model.DecisionAdmittedFirst,
			ResultDepartmentID: 1,
			ResultRoleID:       2,
		}
		err = svc.CreateInterviewResult(ctx, operator.ID, createReq)
		if err == nil || !strings.Contains(err.Error(), "不在时间范围内") {
			t.Fatalf("expected '不在时间范围内' when creating interview result during edit period, got: %v", err)
		}

		// 3. 超出周期范围后查看应被拦截
		outTerm := model.Term{
			ID:             21,
			Title:          "已过期周期",
			Type:           model.TermTypeRecruit,
			Year:           2025,
			EditStartAt:    now.Add(-48 * time.Hour),
			EditEndAt:      now.Add(-24 * time.Hour),
			QueryStartAt:   now.Add(-23 * time.Hour),
			QueryEndAt:     now.Add(-1 * time.Hour),
			ExecuteAfterAt: now.Add(-30 * time.Minute),
		}
		db.Create(&outTerm)
		_, err = svc.GetApplicationList(ctx, operator.ID, 0, outTerm.ID, nil)
		if err == nil || !strings.Contains(err.Error(), "不在查询时间范围内") {
			t.Fatalf("expected '不在查询时间范围内' for expired term, got: %v", err)
		}
	})
}

func TestExportInterviewResults(t *testing.T) {
	svc, db := setupTestTermService(t)
	ctx := context.Background()
	now := time.Now()

	// 准备基础部门与角色数据
	deptDev := model.Department{ID: 10, Name: "开发部", Type: "部门"}
	deptNet := model.Department{ID: 20, Name: "网络部", Type: "部门"}
	db.Create(&deptDev)
	db.Create(&deptNet)

	roleVP := model.Role{ID: 100, Name: "副会长", Level: 80, Type: "理事会", IsActive: true}
	roleMinister := model.Role{ID: 101, Name: "部长", Level: 50, Type: "部门", IsActive: true}
	roleOfficer := model.Role{ID: 102, Name: "干事", Level: 10, Type: "部门", IsActive: true}
	db.Create(&roleVP)
	db.Create(&roleMinister)
	db.Create(&roleOfficer)

	// 准备用户：
	// 1. 副会长 (Level 80, 未毕业)
	vpUser := model.User{
		ID:           500,
		Name:         "副会长张三",
		StudentID:    "20230000500",
		Username:     "vp_user",
		DepartmentID: deptDev.ID,
		RoleID:       roleVP.ID,
		IsGraduated:  false,
	}
	// 2. 部长 (Level 50, 未毕业，权限不够)
	ministerUser := model.User{
		ID:           501,
		Name:         "部长李四",
		StudentID:    "20230000501",
		Username:     "minister_user",
		DepartmentID: deptDev.ID,
		RoleID:       roleMinister.ID,
		IsGraduated:  false,
	}
	// 3. 已毕业副会长 (Level 80, 已毕业，无操作权限)
	graduatedVP := model.User{
		ID:           502,
		Name:         "老副会长王五",
		StudentID:    "20210000502",
		Username:     "grad_vp",
		DepartmentID: deptDev.ID,
		RoleID:       roleVP.ID,
		IsGraduated:  true,
	}
	// 4. 申请人1 (带面试结果)
	candidate1 := model.User{
		ID:           503,
		Name:         "申请人甲",
		StudentID:    "20240101001",
		Username:     "cand_1",
		DepartmentID: deptDev.ID,
		RoleID:       roleOfficer.ID,
		IsGraduated:  false,
	}
	// 5. 申请人2 (无面试结果，待定)
	candidate2 := model.User{
		ID:           504,
		Name:         "申请人乙",
		StudentID:    "20240101002",
		Username:     "cand_2",
		DepartmentID: deptNet.ID,
		RoleID:       roleOfficer.ID,
		IsGraduated:  false,
	}
	db.Create(&vpUser)
	db.Create(&ministerUser)
	db.Create(&graduatedVP)
	db.Create(&candidate1)
	db.Create(&candidate2)

	// 周期：已过期周期 (满足“时间可以不用限制”测试)
	expiredTerm := model.Term{
		ID:             50,
		Title:          "2025年秋季招新",
		Type:           model.TermTypeRecruit,
		Year:           2025,
		EditStartAt:    now.Add(-72 * time.Hour),
		EditEndAt:      now.Add(-48 * time.Hour),
		QueryStartAt:   now.Add(-47 * time.Hour),
		QueryEndAt:     now.Add(-24 * time.Hour),
		ExecuteAfterAt: now.Add(-12 * time.Hour),
	}
	db.Create(&expiredTerm)

	trueVal := true
	app1 := model.Application{
		ID:                 5001,
		TermID:             expiredTerm.ID,
		Type:               model.TermTypeRecruit,
		UserID:             candidate1.ID,
		Name:               candidate1.Name,
		Gender:             "男",
		StudentID:          candidate1.StudentID,
		College:            "计算机科学与工程学院",
		MajorClass:         "计科241",
		PoliticalStatus:    "共青团员",
		BirthDate:          "2005-09",
		Phone:              "13800000001",
		QQ:                 "10001",
		FirstChoice:        model.OrganizationRole{DepartmentID: deptDev.ID, RoleID: roleOfficer.ID},
		SecondChoice:       model.OrganizationRole{DepartmentID: deptNet.ID, RoleID: roleOfficer.ID},
		AllowAdjust:        &trueVal,
		Decision:           model.DecisionAdmittedFirst,
		Result:             model.OrganizationRole{DepartmentID: deptDev.ID, RoleID: roleOfficer.ID},
		OperatorID:         vpUser.ID,
		DecisionRemark:     "表现优异，通过第一志愿",
		CreatedAt:          now.Add(-48 * time.Hour),
		UpdatedAt:          now.Add(-30 * time.Hour),
	}
	app2 := model.Application{
		ID:              5002,
		TermID:          expiredTerm.ID,
		Type:            model.TermTypeRecruit,
		UserID:          candidate2.ID,
		Name:            candidate2.Name,
		Gender:          "女",
		StudentID:       candidate2.StudentID,
		College:         "理学院",
		MajorClass:      "统计241",
		PoliticalStatus: "共青团员",
		BirthDate:       "2005-10",
		Phone:           "13800000002",
		QQ:              "10002",
		FirstChoice:     model.OrganizationRole{DepartmentID: deptNet.ID, RoleID: roleOfficer.ID},
		SecondChoice:    model.OrganizationRole{DepartmentID: deptDev.ID, RoleID: roleOfficer.ID},
		AllowAdjust:     &trueVal,
		Decision:        model.DecisionPending,
		OperatorID:      0,
		CreatedAt:       now.Add(-47 * time.Hour),
		UpdatedAt:       now.Add(-47 * time.Hour),
	}
	db.Create(&app1)
	db.Create(&app2)

	t.Run("权限校验：低于副会长（部长）不能导出", func(t *testing.T) {
		req := request.ExportInterviewResultReq{TermID: expiredTerm.ID}
		_, _, err := svc.ExportInterviewResults(ctx, ministerUser.ID, req)
		if err == nil || !strings.Contains(err.Error(), "权限不够") {
			t.Fatalf("expected '权限不够' for minister, got: %v", err)
		}
	})

	t.Run("权限校验：已毕业副会长不能导出", func(t *testing.T) {
		req := request.ExportInterviewResultReq{TermID: expiredTerm.ID}
		_, _, err := svc.ExportInterviewResults(ctx, graduatedVP.ID, req)
		if err == nil || !strings.Contains(err.Error(), "已毕业成员无操作权限") {
			t.Fatalf("expected '已毕业成员无操作权限' for graduated VP, got: %v", err)
		}
	})

	t.Run("时间不限制：即使周期已过期，在任副会长仍可成功导出Excel", func(t *testing.T) {
		req := request.ExportInterviewResultReq{TermID: expiredTerm.ID}
		bytesData, filename, err := svc.ExportInterviewResults(ctx, vpUser.ID, req)
		if err != nil {
			t.Fatalf("expected export to succeed, got error: %v", err)
		}
		if !strings.Contains(filename, "2025年秋季招新_面试结果.xlsx") {
			t.Fatalf("expected filename containing '2025年秋季招新_面试结果.xlsx', got: %s", filename)
		}
		if len(bytesData) == 0 {
			t.Fatalf("expected non-empty byte slice")
		}

		// 使用 excelize 解析导出的 Excel 文件，验证结构与数据
		excelFile, err := excelize.OpenReader(bytes.NewReader(bytesData))
		if err != nil {
			t.Fatalf("failed to open generated excel: %v", err)
		}
		defer excelFile.Close()

		sheetName := "面试结果"
		rows, err := excelFile.GetRows(sheetName)
		if err != nil {
			t.Fatalf("failed to get rows from sheet %s: %v", sheetName, err)
		}

		// 验证表头与数据行数：1 表头 + 2 申请记录 = 3 行
		if len(rows) != 3 {
			t.Fatalf("expected 3 rows, got %d", len(rows))
		}

		// 验证表头各关键列名
		headers := rows[0]
		expectedHeaders := []string{"业务周期", "业务类型", "学号", "姓名", "性别", "学院", "专业班级"}
		for i, expected := range expectedHeaders {
			if headers[i] != expected {
				t.Fatalf("header col %d: expected %s, got %s", i, expected, headers[i])
			}
		}

		// 验证数据内容：由于按 created_at DESC 排序，app2在前，app1在后
		row1 := rows[1] // app2
		if row1[2] != candidate2.StudentID || row1[3] != candidate2.Name || row1[15] != model.DecisionPending {
			t.Fatalf("row 1 data mismatch: student=%s, name=%s, decision=%s", row1[2], row1[3], row1[15])
		}
		if row1[18] != "-" {
			t.Fatalf("expected operator name '-' for pending app, got %s", row1[18])
		}

		row2 := rows[2] // app1
		if row2[2] != candidate1.StudentID || row2[3] != candidate1.Name || row2[15] != model.DecisionAdmittedFirst {
			t.Fatalf("row 2 data mismatch: student=%s, name=%s, decision=%s", row2[2], row2[3], row2[15])
		}
		if row2[18] != vpUser.Name {
			t.Fatalf("expected operator name %s, got %s", vpUser.Name, row2[18])
		}
	})

	t.Run("支持按部门筛选申请表导出", func(t *testing.T) {
		req := request.ExportInterviewResultReq{
			TermID:       expiredTerm.ID,
			DepartmentID: deptDev.ID,
		}
		bytesData, _, err := svc.ExportInterviewResults(ctx, vpUser.ID, req)
		if err != nil {
			t.Fatalf("export with department filter failed: %v", err)
		}
		excelFile, err := excelize.OpenReader(bytes.NewReader(bytesData))
		if err != nil {
			t.Fatalf("failed to open generated excel: %v", err)
		}
		defer excelFile.Close()

		rows, err := excelFile.GetRows("面试结果")
		if err != nil {
			t.Fatalf("failed to get rows: %v", err)
		}
		if len(rows) < 2 {
			t.Fatalf("expected rows with deptDev, got %d", len(rows))
		}
	})

	t.Run("位状态压缩字段导出测试", func(t *testing.T) {
		// 1. 获取字段定义列表元数据
		meta := svc.GetExportFieldsDefinition()
		if meta.TotalFields != 22 {
			t.Fatalf("expected 22 export field definitions, got %d", meta.TotalFields)
		}
		if meta.AllMask != request.ExportFieldAllMask {
			t.Fatalf("expected all_mask %d, got %d", request.ExportFieldAllMask, meta.AllMask)
		}
		for _, def := range meta.Fields {
			if def.Mask != (uint64(1) << def.Bit) {
				t.Fatalf("def %s: expected mask %d, got %d", def.Key, uint64(1)<<def.Bit, def.Mask)
			}
		}

		// 2. 自定义选定字段导出：使用 010101 状态位序列，仅导出 学号 (Index:2) + 姓名 (Index:3) + 面试决策 (Index:15)
		// 下标：0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21
		// 状态：0 0 1 1 0 0 0 0 0 0  0  0  0  0  0  1  0  0  0  0  0  0
		bitSeq := "0011000000000001000000"

		req := request.ExportInterviewResultReq{
			TermID: expiredTerm.ID,
			Fields: bitSeq,
		}
		bytesData, _, err := svc.ExportInterviewResults(ctx, vpUser.ID, req)
		if err != nil {
			t.Fatalf("export with selected fields failed: %v", err)
		}

		excelFile, err := excelize.OpenReader(bytes.NewReader(bytesData))
		if err != nil {
			t.Fatalf("failed to open generated excel: %v", err)
		}
		defer excelFile.Close()

		rows, err := excelFile.GetRows("面试结果")
		if err != nil {
			t.Fatalf("failed to get rows: %v", err)
		}

		// 验证列数：恰好只有 3 列
		if len(rows[0]) != 3 {
			t.Fatalf("expected 3 header columns, got %d: %v", len(rows[0]), rows[0])
		}
		if rows[0][0] != "学号" || rows[0][1] != "姓名" || rows[0][2] != "面试决策" {
			t.Fatalf("expected headers [学号, 姓名, 面试决策], got %v", rows[0])
		}

		// 验证数据行只有 3 列
		if len(rows[1]) != 3 || rows[1][0] != candidate2.StudentID || rows[1][1] != candidate2.Name || rows[1][2] != model.DecisionPending {
			t.Fatalf("row 1 data mismatch: %v", rows[1])
		}
		if len(rows[2]) != 3 || rows[2][0] != candidate1.StudentID || rows[2][1] != candidate1.Name || rows[2][2] != model.DecisionAdmittedFirst {
			t.Fatalf("row 2 data mismatch: %v", rows[2])
		}

		// 3. 兼容数值型字符串掩码 (32780 = (1<<2)|(1<<3)|(1<<15))
		reqNumeric := request.ExportInterviewResultReq{
			TermID: expiredTerm.ID,
			Fields: "32780",
		}
		bytesDataNum, _, err := svc.ExportInterviewResults(ctx, vpUser.ID, reqNumeric)
		if err != nil {
			t.Fatalf("export with numeric mask string failed: %v", err)
		}
		excelFileNum, err := excelize.OpenReader(bytes.NewReader(bytesDataNum))
		if err != nil {
			t.Fatalf("failed to open generated excel: %v", err)
		}
		defer excelFileNum.Close()
		rowsNum, _ := excelFileNum.GetRows("面试结果")
		if len(rowsNum[0]) != 3 {
			t.Fatalf("expected 3 header columns with numeric mask, got %d", len(rowsNum[0]))
		}

		// 4. 无效掩码（全0状态序列，未选择任何有效字段）测试
		invalidReq := request.ExportInterviewResultReq{
			TermID: expiredTerm.ID,
			Fields: "0000000000000000000000",
		}
		_, _, err = svc.ExportInterviewResults(ctx, vpUser.ID, invalidReq)
		if err == nil || !strings.Contains(err.Error(), "未选择任何有效导出字段") {
			t.Fatalf("expected '未选择任何有效导出字段' error, got: %v", err)
		}

		// 5. 不存在的部门ID筛选测试
		invalidDeptReq := request.ExportInterviewResultReq{
			TermID:       expiredTerm.ID,
			DepartmentID: 999999,
		}
		_, _, err = svc.ExportInterviewResults(ctx, vpUser.ID, invalidDeptReq)
		if err == nil || !strings.Contains(err.Error(), "部门不存在") {
			t.Fatalf("expected '部门不存在' error, got: %v", err)
		}
	})
}

