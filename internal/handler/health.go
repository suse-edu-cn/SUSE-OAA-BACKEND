package handler

import (
	"context"
	"sync"
	"time"

	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHealthHandler(db *gorm.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, rdb: rdb}
}

func (h *HealthHandler) Livez(c *gin.Context) {
	response.Success(c, nil)
}

func (h *HealthHandler) Healthz(c *gin.Context) {

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex
	failures := make(map[string]string)

	wg.Go(func() {
		if h.db != nil {
			sqlDB, err := h.db.DB()
			if err != nil {
				mu.Lock()
				failures["mysql"] = err.Error()
				mu.Unlock()
			} else if err := sqlDB.PingContext(ctx); err != nil {
				mu.Lock()
				failures["mysql"] = err.Error()
				mu.Unlock()
			}
		} else {
			mu.Lock()
			failures["mysql"] = "mysql client not initialized"
			mu.Unlock()
		}
	})

	wg.Go(func() {
		if h.rdb != nil {
			if err := h.rdb.Ping(ctx).Err(); err != nil {
				mu.Lock()
				failures["redis"] = err.Error()
				mu.Unlock()
			}
		} else {
			mu.Lock()
			failures["redis"] = "redis client not initialized"
			mu.Unlock()
		}
	})

	wg.Wait()

	if len(failures) == 0 {
		response.Success(c, nil)
		return
	}

	response.ServiceUnavailable(c, "service unavailable", failures)
}
