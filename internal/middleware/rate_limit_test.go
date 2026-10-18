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
	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			req.RemoteAddr = "192.168.1.100:12345"

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				passCount.Add(1)
			} else if w.Code == http.StatusTooManyRequests {
				blockCount.Add(1)
			}
		}()
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
