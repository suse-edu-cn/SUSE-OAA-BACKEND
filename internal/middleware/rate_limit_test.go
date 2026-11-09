package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRateLimit_SingleRequest(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer rdb.Close()

	r := gin.New()
	r.Use(RateLimit(rdb, "test_single", 2, time.Minute))
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("预期返回 200，实际返回: %d", w.Code)
	}
	t.Logf("第一次请求成功放行，状态码: %d，响应内容: %s", w.Code, w.Body.String())
}

func TestRateLimit_Concurrent(t *testing.T) {

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer rdb.Close()

	const limit = 5
	r := gin.New()
	r.Use(RateLimit(rdb, "test_100_burst", limit, time.Minute))
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	const totalRequests = 1000
	var passCount atomic.Int64
	var blockCount atomic.Int64

	var wg sync.WaitGroup
	for range totalRequests {
		wg.Go(func() {

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			req.RemoteAddr = "192.168.1.100:12345"

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				passCount.Add(1)
			} else if w.Code == http.StatusTooManyRequests {
				blockCount.Add(1)
			}
		})
	}

	wg.Wait()

	t.Logf("100 并发完成！放行通过: %d 次，拦截阻止: %d 次", passCount.Load(), blockCount.Load())

	if passCount.Load() != limit {
		t.Fatalf("预期放行 %d 次，但实际放行了 %d 次（发生超卖漏洞！）", limit, passCount.Load())
	}
	if blockCount.Load() != totalRequests-limit {
		t.Fatalf("预期拦截 %d 次，但实际拦截了 %d 次", totalRequests-limit, blockCount.Load())
	}
}

func TestRateLimit_FallbackOnRedisDown(t *testing.T) {

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer rdb.Close()

	r := gin.New()
	r.Use(RateLimit(rdb, "test_down", 5, time.Minute))
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	mr.Close()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("预期 Redis 宕机时自动降级放行（返回 200），但实际返回了状态码: %d", w.Code)
	}

	t.Logf("容灾演练成功！Redis 宕机时接口成功降级放行，返回状态码: %d", w.Code)
}

func TestGetRealClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name: "Ali-CDN-Real-IP 优先级最高",
			headers: map[string]string{
				"Ali-CDN-Real-IP": "203.0.113.10",
				"X-Real-IP":       "198.51.100.20",
				"X-Forwarded-For": "192.0.2.30, 10.0.0.1",
			},
			remoteAddr: "172.16.0.1:54321",
			expectedIP: "203.0.113.10",
		},
		{
			name: "X-Real-IP 在无 Ali CDN 头时生效",
			headers: map[string]string{
				"X-Real-IP":       "198.51.100.20",
				"X-Forwarded-For": "192.0.2.30, 10.0.0.1",
			},
			remoteAddr: "172.16.0.1:54321",
			expectedIP: "198.51.100.20",
		},
		{
			name: "X-Forwarded-For 取第一个有效 IP",
			headers: map[string]string{
				"X-Forwarded-For": "192.0.2.30, 10.0.0.1, 172.16.0.1",
			},
			remoteAddr: "172.16.0.1:54321",
			expectedIP: "192.0.2.30",
		},
		{
			name: "无效 IP 头自动跳过兜底",
			headers: map[string]string{
				"Ali-CDN-Real-IP": "invalid-ip-string",
				"X-Real-IP":       "another-bad-ip",
				"X-Forwarded-For": "bad-ip, 192.0.2.55",
			},
			remoteAddr: "172.16.0.1:54321",
			expectedIP: "192.0.2.55",
		},
		{
			name:       "无代理头时兜底 ClientIP",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.50:12345",
			expectedIP: "192.168.1.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			c.Request = req

			actual := GetRealClientIP(c)
			if actual != tt.expectedIP {
				t.Fatalf("预期 IP %s，实际得到 %s", tt.expectedIP, actual)
			}
		})
	}
}

func TestRateLimit_BehindCDN(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer rdb.Close()

	const limit = 2
	r := gin.New()
	r.Use(RateLimit(rdb, "cdn_test", limit, time.Minute))
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// 模拟来自同一个真实用户，但通过不同 CDN 回源节点发来的请求
	cdnNodes := []string{
		"120.221.1.1:11111",
		"120.221.1.2:22222",
		"120.221.1.3:33333",
		"120.221.1.4:44444",
	}

	for i, node := range cdnNodes {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = node
		req.Header.Set("Ali-CDN-Real-IP", "1.2.3.4")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if i < limit {
			if w.Code != http.StatusOK {
				t.Fatalf("第 %d 次请求预期放行(200)，实际状态码: %d", i+1, w.Code)
			}
		} else {
			if w.Code != http.StatusTooManyRequests {
				t.Fatalf("第 %d 次请求预期拦截(429)，实际状态码: %d", i+1, w.Code)
			}
		}
	}
}
