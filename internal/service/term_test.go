package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"suseoaa/internal/model"
	"suseoaa/internal/repository"
	"suseoaa/internal/request"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
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
		if err := svc.checkApplicationAvatar(ctx, "avatar/user.png"); err == nil || !strings.Contains(err.Error(), "申请表照片路径错误") {
			t.Fatalf("expected 申请表照片路径错误 for avatar/, got: %v", err)
		}
		if err := svc.checkApplicationAvatar(ctx, "illegal/path.png"); err == nil || !strings.Contains(err.Error(), "申请表照片路径错误") {
			t.Fatalf("expected 申请表照片路径错误, got: %v", err)
		}

		// 2. 路径穿越校验与文件类型校验
		if err := svc.checkApplicationAvatar(ctx, "application/../hack.png"); err == nil || !strings.Contains(err.Error(), "申请表照片路径不合法") {
			t.Fatalf("expected 申请表照片路径不合法, got: %v", err)
		}
		if err := svc.checkApplicationAvatar(ctx, "application/hack.exe"); err == nil || !strings.Contains(err.Error(), "申请表照片类型错误") {
			t.Fatalf("expected 申请表照片类型错误, got: %v", err)
		}

		// 3. 正常路径与空路径允许通过
		if err := svc.checkApplicationAvatar(ctx, "application/pic.png"); err != nil {
			t.Fatalf("valid avatar should be allowed, got: %v", err)
		}
		if err := svc.checkApplicationAvatar(ctx, ""); err != nil {
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
}
