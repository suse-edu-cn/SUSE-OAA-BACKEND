package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"suseoaa/internal/model"
	"suseoaa/internal/repository"
	"suseoaa/internal/service"
	"suseoaa/pkg/response"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestUserHandler(t *testing.T) (*UserHandler, *gorm.DB) {
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
	svc := service.NewUserService(repo, nil, nil, nil, nil)
	return NewUserHandler(svc), db
}

func TestUserHandler_GetUserList_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, db := setupTestUserHandler(t)

	dept := model.Department{ID: 1, Name: "开发部", Type: "部门", IsActive: true}
	_ = db.Create(&dept).Error
	roleMember := model.Role{ID: 1, Name: "普通会员", Level: 10, Type: "部门", IsActive: true}
	_ = db.Create(&roleMember).Error

	lowUser := model.User{
		ID:           101,
		StudentID:    "101",
		Username:     "low_user",
		Name:         "低权限用户",
		DepartmentID: dept.ID,
		RoleID:       roleMember.ID,
		Grade:        2024,
		IsGraduated:  false,
	}
	_ = db.Create(&lowUser).Error

	r := gin.New()
	r.GET("/user/list", func(c *gin.Context) {
		c.Set("user_id", lowUser.ID)
		h.GetUserList(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/user/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for member user, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp response.Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 403 || resp.Message != "权限不足" {
		t.Fatalf("expected code 403 and '权限不足', got code=%d, msg=%q", resp.Code, resp.Message)
	}
}

func TestUserHandler_GetUserList_SuccessForViceMinister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, db := setupTestUserHandler(t)

	dept := model.Department{ID: 2, Name: "秘书处", Type: "部门", IsActive: true}
	_ = db.Create(&dept).Error
	roleVP := model.Role{ID: 2, Name: "副部长", Level: 50, Type: "部门", IsActive: true}
	_ = db.Create(&roleVP).Error

	vpUser := model.User{
		ID:           102,
		StudentID:    "102",
		Username:     "vp_user",
		Name:         "副部长用户",
		DepartmentID: dept.ID,
		RoleID:       roleVP.ID,
		Grade:        2023,
		IsGraduated:  false,
	}
	_ = db.Create(&vpUser).Error

	r := gin.New()
	r.GET("/user/list", func(c *gin.Context) {
		c.Set("user_id", vpUser.ID)
		h.GetUserList(c)
	})

	req, _ := http.NewRequest(http.MethodGet, "/user/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for vice minister, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp response.Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 200 {
		t.Fatalf("expected code 200, got code=%d, msg=%q", resp.Code, resp.Message)
	}
}
