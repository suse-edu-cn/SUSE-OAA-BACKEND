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
	"gorm.io/driver/sqlite"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// setupTestUserService 搭建轻量级内存 SQLite + miniredis 真实环境
func setupTestUserService(t *testing.T) (*UserService, *gorm.DB, *miniredis.Miniredis) {
	// 使用纯 Go SQLite 内存数据库，微秒级执行，不依赖本地 MySQL
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite in-memory db: %v", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.Department{}, &model.Role{}); err != nil {
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

	repo := repository.NewUserRepository(db, rdb)
	svc := NewUserService(repo, nil, nil, nil, nil)
	return svc, db, mr
}

// 1. 测试真实接口：UserService.Register 参数校验
func TestUserService_Register_ValidationTableDriven(t *testing.T) {
	svc, _, _ := setupTestUserService(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		req     request.RegisterReq
		wantErr string
	}{
		{
			name: "空学号",
			req: request.RegisterReq{
				StudentID: "",
				Username:  "zhangsan",
				Name:      "张三",
				Email:     "zhangsan@test.com",
				Password:  "123456",
				Grade:     2024,
			},
			wantErr: "注册信息不能为空",
		},
		{
			name: "纯空格学号",
			req: request.RegisterReq{
				StudentID: "   ",
				Username:  "zhangsan",
				Name:      "张三",
				Email:     "zhangsan@test.com",
				Password:  "123456",
				Grade:     2024,
			},
			wantErr: "注册信息不能为空",
		},
		{
			name: "空密码",
			req: request.RegisterReq{
				StudentID: "20240101",
				Username:  "zhangsan",
				Name:      "张三",
				Email:     "zhangsan@test.com",
				Password:  "",
				Grade:     2024,
			},
			wantErr: "注册信息不能为空",
		},
		{
			name: "入学年份为0",
			req: request.RegisterReq{
				StudentID: "20240101",
				Username:  "zhangsan",
				Name:      "张三",
				Email:     "zhangsan@test.com",
				Password:  "123456",
				Grade:     0,
			},
			wantErr: "入学年份不合法",
		},
		{
			name: "入学年份等于2000",
			req: request.RegisterReq{
				StudentID: "20240101",
				Username:  "zhangsan",
				Name:      "张三",
				Email:     "zhangsan@test.com",
				Password:  "123456",
				Grade:     2000,
			},
			wantErr: "入学年份不合法",
		},
		{
			name: "入学年份大于9999",
			req: request.RegisterReq{
				StudentID: "20240101",
				Username:  "zhangsan",
				Name:      "张三",
				Email:     "zhangsan@test.com",
				Password:  "123456",
				Grade:     10000,
			},
			wantErr: "入学年份不合法",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.Register(ctx, tc.req)
			if err == nil {
				t.Fatalf("expected error %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

// 2. 测试真实接口：UserService.UpdatePassword 修改密码逻辑
func TestUserService_UpdatePassword_RealLogic(t *testing.T) {
	svc, db, _ := setupTestUserService(t)
	ctx := context.Background()

	// 准备初始用户数据
	initialPass := "OldSecret123"
	hashedPass, _ := bcrypt.GenerateFromPassword([]byte(initialPass), bcrypt.DefaultCost)
	testUser := model.User{
		StudentID: "20240001",
		Username:  "user_pass_test",
		Name:      "测试用户",
		Password:  string(hashedPass),
	}
	if err := db.Create(&testUser).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Case 1: 用户不存在，预期返回 "用户不存在"
	err := svc.UpdatePassword(ctx, 99999, initialPass, "NewPass456")
	if err == nil || err.Error() != "用户不存在" {
		t.Fatalf("expected '用户不存在', got: %v", err)
	}

	// Case 2: 旧密码输入错误，预期返回 "旧密码错误"
	err = svc.UpdatePassword(ctx, testUser.ID, "WrongOldPass", "NewPass456")
	if err == nil || err.Error() != "旧密码错误" {
		t.Fatalf("expected '旧密码错误', got: %v", err)
	}

	// Case 3: 输入正确的旧密码，修改成功返回 nil
	err = svc.UpdatePassword(ctx, testUser.ID, initialPass, "NewPass456")
	if err != nil {
		t.Fatalf("expected update password success, got: %v", err)
	}

	// 深入验证：从数据库重新查询，验证密码字段确实已被更新，且能用新密码校验通过！
	var updatedUser model.User
	db.First(&updatedUser, testUser.ID)
	if err := bcrypt.CompareHashAndPassword([]byte(updatedUser.Password), []byte("NewPass456")); err != nil {
		t.Fatalf("expected new password to match in database, got error: %v", err)
	}
}

// 3. 测试真实接口：UserService.CancelDeleteUser 取消注销冷静期
func TestUserService_CancelDeleteUser_RealLogic(t *testing.T) {
	svc, db, _ := setupTestUserService(t)
	ctx := context.Background()

	now := time.Now()
	testUser := model.User{
		StudentID: "20240002",
		Username:  "user_cancel_delete",
		Name:      "注销测试",
	}
	db.Create(&testUser)

	// Case 1: 账号未处于注销冷静期（ScheduledDeleteAt 为 nil）
	err := svc.CancelDeleteUser(ctx, testUser.ID, "123456")
	if err == nil || err.Error() != "当前账号未处于注销冷静期" {
		t.Fatalf("expected '当前账号未处于注销冷静期', got: %v", err)
	}

	// Case 2: 冷静期已过期（ScheduledDeleteAt 在过去）
	pastTime := now.Add(-10 * time.Minute)
	db.Model(&testUser).Update("scheduled_delete_at", pastTime)
	err = svc.CancelDeleteUser(ctx, testUser.ID, "123456")
	if err == nil || err.Error() != "注销冷静期已结束，无法取消" {
		t.Fatalf("expected '注销冷静期已结束，无法取消', got: %v", err)
	}

	// Case 3: 处于有效冷静期内，输入错误验证码
	futureTime := now.Add(24 * time.Hour)
	db.Model(&testUser).Update("scheduled_delete_at", futureTime)
	err = svc.CancelDeleteUser(ctx, testUser.ID, "wrong_code")
	if err == nil {
		t.Fatalf("expected error for wrong code, got nil")
	}

	// Case 4: 处于有效冷静期内，输入正确验证码 -> 成功恢复！
	const validCode = "888666"
	_ = svc.Repo.SaveVerificationCode(ctx, testUser.ID, validCode, "cancel_delete", 5*time.Minute)
	err = svc.CancelDeleteUser(ctx, testUser.ID, validCode)
	if err != nil {
		t.Fatalf("expected cancel delete success, got: %v", err)
	}

	// 深入验证：数据库中的 scheduled_delete_at 应该被重置为 NULL
	var restoredUser model.User
	db.First(&restoredUser, testUser.ID)
	if restoredUser.ScheduledDeleteAt != nil {
		t.Fatalf("expected scheduled_delete_at to be reset to nil, got: %v", restoredUser.ScheduledDeleteAt)
	}
}

// 4. 测试真实接口：UserService.DeleteUser 参数互斥与本人越权防御
func TestUserService_DeleteUser_ParameterExclusion(t *testing.T) {
	svc, _, _ := setupTestUserService(t)
	ctx := context.Background()

	// Case 1: user_id 和 code 严禁同时传参
	_, err := svc.DeleteUser(ctx, 1001, 1002, "123456")
	if err == nil || err.Error() != "参数错误：user_id 与 code 不能同时传递" {
		t.Fatalf("expected mutual exclusion error, got: %v", err)
	}

	// Case 2: 管理员不能通过传 user_id==id 走管理员通道注销自己
	_, err = svc.DeleteUser(ctx, 1001, 1001, "")
	if err == nil || err.Error() != "注销本人账号请提供验证码，或留空查询冷静期" {
		t.Fatalf("expected self-deletion prevention error, got: %v", err)
	}
}

// 5. 测试真实接口：UserService.GetUserInfo 判断账号是否处于注销冷静期
func TestUserService_GetUserInfo_DeletionCooldown(t *testing.T) {
	svc, db, _ := setupTestUserService(t)
	ctx := context.Background()

	future := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	past := time.Now().Add(-1 * time.Hour).Truncate(time.Second)

	userNormal := model.User{
		ID:        2001,
		StudentID: "2024001",
		Username:  "user2001",
		Name:      "正常用户",
	}
	userDeleting := model.User{
		ID:                2002,
		StudentID:         "2024002",
		Username:  "user2002",
		Name:              "冷静期用户",
		ScheduledDeleteAt: &future,
	}
	userExpired := model.User{
		ID:                2003,
		StudentID:         "2024003",
		Username:  "user2003",
		Name:              "过期待清理用户",
		ScheduledDeleteAt: &past,
	}

	for _, u := range []model.User{userNormal, userDeleting, userExpired} {
		if err := db.Create(&u).Error; err != nil {
			t.Fatalf("failed to insert test user: %v", err)
		}
	}

	// 1. 正常用户
	infoNormal, err := svc.GetUserInfo(ctx, userNormal.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if infoNormal.IsDeleting {
		t.Fatalf("expected is_deleting=false, got true")
	}
	if infoNormal.ScheduledDeleteAt != "" {
		t.Fatalf("expected scheduled_delete_at=\"\", got %v", infoNormal.ScheduledDeleteAt)
	}

	// 2. 冷静期用户 (当前时间在 ScheduledDeleteAt 之前)
	infoDeleting, err := svc.GetUserInfo(ctx, userDeleting.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !infoDeleting.IsDeleting {
		t.Fatalf("expected is_deleting=true, got false")
	}
	expectedFormat := future.In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format("2006-01-02 15:04:05")
	if infoDeleting.ScheduledDeleteAt != expectedFormat {
		t.Fatalf("expected scheduled_delete_at=%s, got %s", expectedFormat, infoDeleting.ScheduledDeleteAt)
	}

	// 3. 过期用户 (ScheduledDeleteAt 已过去，不再属于有效冷静期)
	infoExpired, err := svc.GetUserInfo(ctx, userExpired.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if infoExpired.IsDeleting {
		t.Fatalf("expected is_deleting=false for expired delete, got true")
	}
	if infoExpired.ScheduledDeleteAt != "" {
		t.Fatalf("expected scheduled_delete_at=\"\" for expired delete, got %v", infoExpired.ScheduledDeleteAt)
	}
}

// 6. 测试真实接口：UserService.BatchUserInfo 毕业标记与毕业人员不可任命在任职务
func TestUserService_BatchUserInfo_Graduation(t *testing.T) {
	svc, db, _ := setupTestUserService(t)
	svc.RoleRepo = repository.NewRoleRepository(db)
	svc.DepartmentRepo = repository.NewDepartmentRepository(db)
	ctx := context.Background()

	// 初始化角色与部门
	roleDev := model.Role{ID: 1, Name: "开发者", Level: 100, Type: "协会", IsActive: true}
	roleVice := model.Role{ID: 3, Name: "副会长", Level: 80, Type: "理事会", IsActive: true}
	roleMinister := model.Role{ID: 4, Name: "部长", Level: 60, Type: "部门", IsActive: true}
	roleClerk := model.Role{ID: 5, Name: "干事", Level: 20, Type: "部门", IsActive: true}
	roleMember := model.Role{ID: 7, Name: "会员", Level: 10, Type: "协会", IsActive: true}
	for _, r := range []model.Role{roleDev, roleVice, roleMinister, roleClerk, roleMember} {
		_ = db.Create(&r).Error
	}

	deptAssoc := model.Department{ID: 6, Name: "开放原子开源协会", Type: "协会", IsActive: true}
	deptAlgo := model.Department{ID: 1, Name: "算法竞赛部", Type: "部门", IsActive: true}
	for _, d := range []model.Department{deptAssoc, deptAlgo} {
		_ = db.Create(&d).Error
	}

	// 1. 创建测试用户
	// 操作者：副会长 (ID: 3000)
	opUser := model.User{ID: 3000, StudentID: "op001", Username: "op", Name: "操作人", RoleID: roleVice.ID, DepartmentID: deptAssoc.ID}
	// 在任部长：(ID: 3001)
	activeMinister := model.User{ID: 3001, StudentID: "stu001", Username: "minister", Name: "在任部长", RoleID: roleMinister.ID, DepartmentID: deptAlgo.ID, Grade: 2022, IsGraduated: false}
	// 已毕业人员：(ID: 3002)
	gradUser := model.User{ID: 3002, StudentID: "stu002", Username: "grad", Name: "老学长", RoleID: roleMember.ID, DepartmentID: deptAssoc.ID, Grade: 2020, IsGraduated: true}
	for _, u := range []model.User{opUser, activeMinister, gradUser} {
		_ = db.Create(&u).Error
	}

	// Case A: 尝试将已毕业人员 (ID: 3002) 任命为算法竞赛部部长 -> 必须被拦截！
	batchReq1 := []request.BatchUserInfoReq{
		{
			UserID:       gradUser.ID,
			DepartmentID: deptAlgo.ID,
			RoleID:       roleMinister.ID,
		},
	}
	res, err := svc.BatchUserInfo(ctx, batchReq1, opUser.DepartmentID, opUser.RoleID)
	if err == nil {
		t.Fatalf("expected error for appointing graduated user, got nil")
	}
	if len(res) == 0 || res[0].ErrorMessage != "已毕业成员不可任命在任职务" {
		t.Fatalf("expected '已毕业成员不可任命在任职务', got: %v", res)
	}

	// Case B: 普通部长 (Level 60) 尝试修改成员的年级 -> 必须被权限锁拦截！
	opMinister := model.User{ID: 3005, StudentID: "op_min", Username: "op_min", Name: "操作部长", RoleID: roleMinister.ID, DepartmentID: deptAlgo.ID}
	targetClerk := model.User{ID: 3006, StudentID: "clerk", Username: "clerk", Name: "干事", RoleID: roleClerk.ID, DepartmentID: deptAlgo.ID}
	_ = db.Create(&opMinister).Error
	_ = db.Create(&targetClerk).Error

	newGrade := uint(2023)
	batchReq2 := []request.BatchUserInfoReq{
		{
			UserID:       targetClerk.ID,
			DepartmentID: deptAlgo.ID,
			RoleID:       roleClerk.ID,
			Grade:        &newGrade,
		},
	}
	res2, err := svc.BatchUserInfo(ctx, batchReq2, opMinister.DepartmentID, opMinister.RoleID)
	if err == nil {
		t.Fatalf("expected permission error for minister modifying grade, got nil")
	}
	if len(res2) == 0 || res2[0].ErrorMessage != "权限不足：只有副会长及以上管理才可修改年级" {
		t.Fatalf("expected '权限不足：只有副会长及以上管理才可修改年级', got: %v", res2)
	}

	// Case C: 副会长 (Level 80) 修改年级 -> 成功
	res3, err := svc.BatchUserInfo(ctx, batchReq2, opUser.DepartmentID, opUser.RoleID)
	if err != nil {
		t.Fatalf("unexpected error for VP modifying grade: %v, res: %v", err, res3)
	}
	var updatedClerk model.User
	if err := db.First(&updatedClerk, targetClerk.ID).Error; err != nil {
		t.Fatalf("failed to query updated clerk: %v", err)
	}
	if updatedClerk.Grade != 2023 {
		t.Fatalf("expected clerk grade to be 2023, got %d", updatedClerk.Grade)
	}
}

// 7. 测试自动毕业 Worker：满 4 年的 6 月份自动标记为毕业
func TestUserService_AutomaticGraduationWorker(t *testing.T) {
	svc, db, _ := setupTestUserService(t)
	ctx := context.Background()

	roleRepo := repository.NewRoleRepository(db)
	deptRepo := repository.NewDepartmentRepository(db)
	svc.RoleRepo = roleRepo
	svc.DepartmentRepo = deptRepo

	roleDev := model.Role{ID: 1, Name: "开发者", Level: 100, Type: "理事会", IsActive: true}
	roleMinister := model.Role{ID: 4, Name: "部长", Level: 60, Type: "部门", IsActive: true}
	roleMember := model.Role{ID: 7, Name: "会员", Level: 10, Type: "协会", IsActive: true}
	for _, r := range []model.Role{roleDev, roleMinister, roleMember} {
		_ = db.Create(&r).Error
	}

	deptAssoc := model.Department{ID: 6, Name: "开放原子开源协会", Type: "协会", IsActive: true}
	deptAlgo := model.Department{ID: 1, Name: "算法竞赛部", Type: "部门", IsActive: true}
	for _, d := range []model.Department{deptAssoc, deptAlgo} {
		_ = db.Create(&d).Error
	}

	// A. 测试年份计算算法 CalculateGraduationMaxGrade
	t.Run("毕业年份临界值算法校验", func(t *testing.T) {
		loc := time.FixedZone("Asia/Shanghai", 8*60*60)
		// 2026年10月（>= 6月） -> 2022级毕业
		t1 := time.Date(2026, 10, 29, 12, 0, 0, 0, loc)
		if g := svc.CalculateGraduationMaxGrade(t1); g != 2022 {
			t.Fatalf("expected 2022, got %d", g)
		}
		// 2026年6月1日（正好6月） -> 2022级毕业
		t2 := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
		if g := svc.CalculateGraduationMaxGrade(t2); g != 2022 {
			t.Fatalf("expected 2022, got %d", g)
		}
		// 2026年5月31日（< 6月） -> 2021级毕业，2022级尚未毕业
		t3 := time.Date(2026, 5, 31, 23, 59, 59, 0, loc)
		if g := svc.CalculateGraduationMaxGrade(t3); g != 2021 {
			t.Fatalf("expected 2021, got %d", g)
		}
		// 2027年1月15日（< 6月） -> 2022级毕业，2023级尚未毕业
		t4 := time.Date(2027, 1, 15, 0, 0, 0, 0, loc)
		if g := svc.CalculateGraduationMaxGrade(t4); g != 2022 {
			t.Fatalf("expected 2022, got %d", g)
		}
	})

	// B. 测试批量毕业 Worker 执行
	t.Run("定时任务批量毕业用户", func(t *testing.T) {
		// 1. 2022级部长（到期应毕业，职务降为会员，归入协会）
		u1 := model.User{ID: 5001, StudentID: "5001", Username: "u1", Name: "22级部长", RoleID: roleMinister.ID, DepartmentID: deptAlgo.ID, Grade: 2022, IsGraduated: false}
		// 2. 2021级开发者（到期应毕业，但保留开发者与原部门）
		u2 := model.User{ID: 5002, StudentID: "5002", Username: "u2", Name: "21级开发者", RoleID: roleDev.ID, DepartmentID: deptAlgo.ID, Grade: 2021, IsGraduated: false}
		// 3. 2024级在任部长（尚未毕业，应保持原样）
		u3 := model.User{ID: 5003, StudentID: "5003", Username: "u3", Name: "24级在任部长", RoleID: roleMinister.ID, DepartmentID: deptAlgo.ID, Grade: 2024, IsGraduated: false}
		for _, u := range []model.User{u1, u2, u3} {
			_ = db.Create(&u).Error
		}

		// 模拟执行：时间为 2026年10月（满 4 年的 2022 级及以前应毕业）
		loc := time.FixedZone("Asia/Shanghai", 8*60*60)
		simNow := time.Date(2026, 10, 29, 0, 0, 0, 0, loc)
		count, err := svc.ExecuteDueUserGraduations(ctx, simNow)
		if err != nil {
			t.Fatalf("unexpected error executing graduations: %v", err)
		}
		if count != 2 {
			t.Fatalf("expected 2 users to graduate, got %d", count)
		}

		// 验证 u1 (2022级部长): is_graduated=true, role=会员, dept=开放原子开源协会
		var user1 model.User
		_ = db.First(&user1, u1.ID)
		if !user1.IsGraduated {
			t.Fatalf("expected user1 to be graduated")
		}
		if user1.RoleID != roleMember.ID {
			t.Fatalf("expected user1 role to be reset to Member (%d), got %d", roleMember.ID, user1.RoleID)
		}
		if user1.DepartmentID != deptAssoc.ID {
			t.Fatalf("expected user1 dept to be reset to Assoc (%d), got %d", deptAssoc.ID, user1.DepartmentID)
		}

		// 验证 u2 (2021级开发者): is_graduated=true, role=开发者, dept=算法竞赛部（保持不变）
		var user2 model.User
		_ = db.First(&user2, u2.ID)
		if !user2.IsGraduated {
			t.Fatalf("expected user2 to be graduated")
		}
		if user2.RoleID != roleDev.ID {
			t.Fatalf("expected developer role to be preserved (%d), got %d", roleDev.ID, user2.RoleID)
		}
		if user2.DepartmentID != deptAlgo.ID {
			t.Fatalf("expected developer dept to be preserved (%d), got %d", deptAlgo.ID, user2.DepartmentID)
		}

		// 验证 u3 (2024级在校生): is_graduated=false, 职务与部门完全不变
		var user3 model.User
		_ = db.First(&user3, u3.ID)
		if user3.IsGraduated {
			t.Fatalf("expected user3 to remain active (not graduated)")
		}
		if user3.RoleID != roleMinister.ID || user3.DepartmentID != deptAlgo.ID {
			t.Fatalf("expected user3 role and dept unchanged, got role=%d dept=%d", user3.RoleID, user3.DepartmentID)
		}
	})
}

// 7. 测试真实接口：UserService.UpdateUserInfo 修改个人资料与年级
func TestUserService_UpdateUserInfo_GradeValidation(t *testing.T) {
	svc, db, _ := setupTestUserService(t)
	ctx := context.Background()

	testUser := model.User{
		ID:        4001,
		StudentID: "4001",
		Username:  "update_user",
		Name:      "测试更新",
		Email:     "update@test.com",
		Grade:     2022,
	}
	_ = db.Create(&testUser).Error

	// 1. 输入不合法入学年份（<= 2000）
	err := svc.UpdateUserInfo(ctx, testUser.ID, "update_user", "update@test.com", "avatar/default.png", 2000)
	if err == nil || err.Error() != "入学年份不合法" {
		t.Fatalf("expected '入学年份不合法', got: %v", err)
	}

	// 2. 输入不合法入学年份（> 9999）
	err = svc.UpdateUserInfo(ctx, testUser.ID, "update_user", "update@test.com", "avatar/default.png", 10000)
	if err == nil || err.Error() != "入学年份不合法" {
		t.Fatalf("expected '入学年份不合法', got: %v", err)
	}
}

// 8. 测试真实接口：UserService.GetUserList 权限控制
func TestUserService_GetUserList_Permission(t *testing.T) {
	svc, db, _ := setupTestUserService(t)
	ctx := context.Background()

	deptDev := model.Department{ID: 10, Name: "开发部", Type: "部门", IsActive: true}
	_ = db.Create(&deptDev).Error

	roleMember := model.Role{ID: 10, Name: "会员", Level: 10, Type: "部门", IsActive: true}
	roleViceMinister := model.Role{ID: 20, Name: "副部长", Level: 50, Type: "部门", IsActive: true}
	roleMinister := model.Role{ID: 30, Name: "部长", Level: 60, Type: "部门", IsActive: true}
	roleInactive := model.Role{ID: 40, Name: "停用职位", Level: 60, Type: "部门", IsActive: false}
	_ = db.Create(&roleMember).Error
	_ = db.Create(&roleViceMinister).Error
	_ = db.Create(&roleMinister).Error
	_ = db.Create(&roleInactive).Error
	_ = db.Model(&model.Role{}).Where("id = ?", roleInactive.ID).Update("is_active", false).Error

	userMember := model.User{
		ID:           5001,
		StudentID:    "5001",
		Username:     "member",
		Name:         "普通会员",
		DepartmentID: deptDev.ID,
		RoleID:       roleMember.ID,
		Grade:        2024,
		IsGraduated:  false,
	}
	userViceMinister := model.User{
		ID:           5002,
		StudentID:    "5002",
		Username:     "vice_minister",
		Name:         "副部长",
		DepartmentID: deptDev.ID,
		RoleID:       roleViceMinister.ID,
		Grade:        2023,
		IsGraduated:  false,
	}
	userMinister := model.User{
		ID:           5003,
		StudentID:    "5003",
		Username:     "minister",
		Name:         "部长",
		DepartmentID: deptDev.ID,
		RoleID:       roleMinister.ID,
		Grade:        2022,
		IsGraduated:  false,
	}
	userGraduated := model.User{
		ID:           5004,
		StudentID:    "5004",
		Username:     "graduated_vp",
		Name:         "已毕业高权限",
		DepartmentID: deptDev.ID,
		RoleID:       roleMinister.ID,
		Grade:        2019,
		IsGraduated:  true,
	}
	userInactive := model.User{
		ID:           5005,
		StudentID:    "5005",
		Username:     "inactive_user",
		Name:         "停用职务用户",
		DepartmentID: deptDev.ID,
		RoleID:       roleInactive.ID,
		Grade:        2023,
		IsGraduated:  false,
	}
	_ = db.Create(&userMember).Error
	_ = db.Create(&userViceMinister).Error
	_ = db.Create(&userMinister).Error
	_ = db.Create(&userGraduated).Error
	_ = db.Create(&userInactive).Error

	t.Run("普通会员(Level 10)应被拦截", func(t *testing.T) {
		_, _, err := svc.GetUserList(ctx, "", "", "", 0, 0, 0, nil, 1, 20, nil, userMember.ID)
		if err == nil || err.Error() != "权限不够" {
			t.Fatalf("expected '权限不够', got: %v", err)
		}
	})

	t.Run("已毕业成员即使曾为部长也应被拦截", func(t *testing.T) {
		_, _, err := svc.GetUserList(ctx, "", "", "", 0, 0, 0, nil, 1, 20, nil, userGraduated.ID)
		if err == nil || err.Error() != "已毕业成员无操作权限" {
			t.Fatalf("expected '已毕业成员无操作权限', got: %v", err)
		}
	})

	t.Run("职位被停用的用户应被拦截", func(t *testing.T) {
		_, _, err := svc.GetUserList(ctx, "", "", "", 0, 0, 0, nil, 1, 20, nil, userInactive.ID)
		if err == nil || err.Error() != "权限不够" {
			t.Fatalf("expected '权限不够', got: %v", err)
		}
	})

	t.Run("副部长(Level 50)允许获取用户列表", func(t *testing.T) {
		list, total, err := svc.GetUserList(ctx, "", "", "", 0, 0, 0, nil, 1, 20, nil, userViceMinister.ID)
		if err != nil {
			t.Fatalf("unexpected error for vice minister: %v", err)
		}
		if total < 5 || len(list) == 0 {
			t.Fatalf("expected users returned, got total=%d, len=%d", total, len(list))
		}
	})

	t.Run("部长(Level 60)允许获取用户列表", func(t *testing.T) {
		list, total, err := svc.GetUserList(ctx, "", "", "", 0, 0, 0, nil, 1, 20, nil, userMinister.ID)
		if err != nil {
			t.Fatalf("unexpected error for minister: %v", err)
		}
		if total < 5 || len(list) == 0 {
			t.Fatalf("expected users returned, got total=%d, len=%d", total, len(list))
		}
	})
}

