package utils

import (
	"testing"
)

// 全局变量用于接收返回值，防止 Go 编译器的死代码消除（Dead Code Elimination）优化掉压测逻辑
var (
	sinkToken  string
	sinkClaims *MyClaims
	sinkErr    error
)

// BenchmarkGenerateToken 测试 JWT 签发（Token 生成）性能
func BenchmarkGenerateToken(b *testing.B) {
	secret := "benchmark-secret-key-12345"
	name := "BenchUser"
	userID := uint64(10086)
	studentID := "20240999"
	expireMinute := 60

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkToken, sinkErr = GenerateToken(name, userID, studentID, secret, expireMinute)
	}
}

// BenchmarkParseToken 测试单核/单协程下 JWT 验签（Token 解析与合法性校验）性能
func BenchmarkParseToken(b *testing.B) {
	secret := "benchmark-secret-key-12345"
	tokenStr, err := GenerateToken("BenchUser", 10086, "20240999", secret, 60)
	if err != nil {
		b.Fatalf("failed to generate token for benchmark: %v", err)
	}

	b.ReportAllocs()
	// 重置计时器：排除 GenerateToken 的准备时间，仅对 ParseToken 进行性能统计
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkClaims, sinkErr = ParseToken(tokenStr, secret)
	}
}

// BenchmarkParseToken_Parallel 测试多协程并发验签性能（模拟真实 Web 服务器高并发请求中间件鉴权场景）
func BenchmarkParseToken_Parallel(b *testing.B) {
	secret := "benchmark-secret-key-12345"
	tokenStr, err := GenerateToken("BenchUser", 10086, "20240999", secret, 60)
	if err != nil {
		b.Fatalf("failed to generate token for benchmark: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = ParseToken(tokenStr, secret)
		}
	})
}
