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
			},
			wantErr: "注册信息不能为空",
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

