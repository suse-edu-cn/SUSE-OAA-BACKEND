package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupTestUserRepo(t *testing.T) (*UserRepository, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() {
		_ = rdb.Close()
	})

	repo := NewUserRepository(nil, rdb)
	return repo, mr
}

// 1. 测试验证码正确校验并消费（防重放）
func TestVerifyAndConsume_Success(t *testing.T) {
	repo, mr := setupTestUserRepo(t)
	ctx := context.Background()
	const userID = uint64(1001)
	const scene = "reset_password"
	const validCode = "654321"

	// 存入有效验证码
	err := repo.SaveVerificationCode(ctx, userID, validCode, scene, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to save code: %v", err)
	}

	// 首次校验正确验证码：必须成功
	err = repo.VerifyAndConsumeVerificationCode(ctx, userID, scene, validCode, 5)
	if err != nil {
		t.Fatalf("expected verify success, got: %v", err)
	}

	// 再次校验相同验证码：必须失败（已消费，防重放）
	err = repo.VerifyAndConsumeVerificationCode(ctx, userID, scene, validCode, 5)
	if err == nil || err.Error() != "验证码错误或已失效" {
		t.Fatalf("expected '验证码错误或已失效' on second consume, got: %v", err)
	}

	// 验证 Redis 中对应 Key 确实不存在
	codeKey := "1001-reset_passwordVerificationCode"
	if mr.Exists(codeKey) {
		t.Fatalf("expected code key to be deleted from redis")
	}
}

// 2. 测试输错 1~4 次累计计数，验证码保留
func TestVerifyAndConsume_WrongCodeCount(t *testing.T) {
	repo, mr := setupTestUserRepo(t)
	ctx := context.Background()
	const userID = uint64(1002)
	const scene = "delete_user"
	const validCode = "123456"

	_ = repo.SaveVerificationCode(ctx, userID, validCode, scene, 5*time.Minute)

	// 连续输错 3 次
	for i := 1; i <= 3; i++ {
		err := repo.VerifyAndConsumeVerificationCode(ctx, userID, scene, "000000", 5)
		if err == nil || err.Error() != "验证码错误" {
			t.Fatalf("attempt %d: expected '验证码错误', got: %v", i, err)
		}
	}

	// 校验失败计数器为 3
	failKey := "1002-delete_userVerificationCode-Fails"
	fails, _ := mr.Get(failKey)
	if fails != "3" {
		t.Fatalf("expected fails count 3, got: %s", fails)
	}

	// 第 4 次输入正确验证码：依然允许通过！
	err := repo.VerifyAndConsumeVerificationCode(ctx, userID, scene, validCode, 5)
	if err != nil {
		t.Fatalf("expected success with correct code after 3 fails, got: %v", err)
	}
}

// 3. 测试输错 5 次触发熔断作废
func TestVerifyAndConsume_MaxFailsFuse(t *testing.T) {
	repo, mr := setupTestUserRepo(t)
	ctx := context.Background()
	const userID = uint64(1003)
	const scene = "reset_password"
	const validCode = "888888"

	_ = repo.SaveVerificationCode(ctx, userID, validCode, scene, 5*time.Minute)

	// 前 4 次输错
	for i := 1; i <= 4; i++ {
		err := repo.VerifyAndConsumeVerificationCode(ctx, userID, scene, "wrong", 5)
		if err == nil || err.Error() != "验证码错误" {
			t.Fatalf("attempt %d: expected '验证码错误', got: %v", i, err)
		}
	}

	// 第 5 次输错：触发熔断作废
	err := repo.VerifyAndConsumeVerificationCode(ctx, userID, scene, "wrong", 5)
	if err == nil || err.Error() != "验证码错误次数过多，已作废，请重新获取" {
		t.Fatalf("attempt 5: expected fuse error, got: %v", err)
	}

	// 验证 Redis 中的验证码已被彻底物理删除
	codeKey := "1003-reset_passwordVerificationCode"
	if mr.Exists(codeKey) {
		t.Fatalf("expected verification code to be deleted after 5 failed attempts")
	}

	// 哪怕此时再输入原真实密码，也无法通过
	err = repo.VerifyAndConsumeVerificationCode(ctx, userID, scene, validCode, 5)
	if err == nil || err.Error() != "验证码错误或已失效" {
		t.Fatalf("expected '验证码错误或已失效' after fuse, got: %v", err)
	}
}

// 4. 测试基于 Redis Lua 脚本的 CAS 冷却锁安全释放
func TestCooldown_LuaCASRelease(t *testing.T) {
	repo, mr := setupTestUserRepo(t)
	ctx := context.Background()
	const userID = uint64(1004)
	const tokenA = "uuid-token-a-1111"
	const tokenB = "uuid-token-b-2222"

	// 1. 请求 A 成功获取冷却锁
	acquired, err := repo.SetCooldownNX(ctx, userID, tokenA, time.Minute)
	if err != nil || !acquired {
		t.Fatalf("expected acquire cooldown success, got: %v, %v", acquired, err)
	}

	// 2. 冷却期内再次申请，必须被拒绝（防并发穿透）
	acquired, err = repo.SetCooldownNX(ctx, userID, tokenB, time.Minute)
	if err != nil || acquired {
		t.Fatalf("expected acquire cooldown rejected during cooldown, got: %v", acquired)
	}

	// 3. 模拟请求 B（拿着错误的 tokenB）尝试删除请求 A 的锁：必须被 Lua CAS 拦截，拒绝删除！
	err = repo.DeleteCooldown(ctx, tokenB, userID)
	if err != nil {
		t.Fatalf("unexpected lua error: %v", err)
	}
	cooldownKey := "1004-CoolDown"
	if !mr.Exists(cooldownKey) {
		t.Fatalf("lock was incorrectly deleted by wrong token!")
	}

	// 4. 只有请求 A（拿着正确的 tokenA）才能真正释放冷却锁！
	err = repo.DeleteCooldown(ctx, tokenA, userID)
	if err != nil {
		t.Fatalf("unexpected lua error: %v", err)
	}
	if mr.Exists(cooldownKey) {
		t.Fatalf("lock should have been deleted by correct token")
	}

	// 5. 释放后，可以再次正常加锁
	acquired, err = repo.SetCooldownNX(ctx, userID, tokenB, time.Minute)
	if err != nil || !acquired {
		t.Fatalf("expected acquire success after release, got: %v, %v", acquired, err)
	}
}
