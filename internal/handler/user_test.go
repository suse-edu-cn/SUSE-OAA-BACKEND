package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	roleRepo := repository.NewRoleRepository(db)
	deptRepo := repository.NewDepartmentRepository(db)
	svc := service.NewUserService(repo, roleRepo, deptRepo, nil, nil)
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

func TestUserHandler_BatchUserInfo_ForbiddenForMember(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, db := setupTestUserHandler(t)

	dept := model.Department{ID: 3, Name: "宣传部", Type: "部门", IsActive: true}
	_ = db.Create(&dept).Error
	roleMember := model.Role{ID: 10, Name: "会员", Level: 10, Type: "部门", IsActive: true}
	_ = db.Create(&roleMember).Error
	roleVP := model.Role{ID: 11, Name: "副会长", Level: 80, Type: "理事会", IsActive: true}
	_ = db.Create(&roleVP).Error

	memberUser := model.User{
		ID:           201,
		StudentID:    "201",
		Username:     "member_user",
		Name:         "普通会员",
		DepartmentID: dept.ID,
		RoleID:       roleMember.ID,
		IsGraduated:  false,
	}
	_ = db.Create(&memberUser).Error

	r := gin.New()
	r.POST("/user/batch", func(c *gin.Context) {
		c.Set("user_id", memberUser.ID)
		h.BatchUserInfo(c)
	})

	body := strings.NewReader(`[{"user_id":201,"department_id":3,"role_id":10}]`)
	req, _ := http.NewRequest(http.MethodPost, "/user/batch", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for member user calling batch, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_BatchUserInfo_ForbiddenForGraduated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, db := setupTestUserHandler(t)

	dept := model.Department{ID: 4, Name: "理事会", Type: "理事会", IsActive: true}
	_ = db.Create(&dept).Error
	roleVP := model.Role{ID: 12, Name: "副会长", Level: 80, Type: "理事会", IsActive: true}
	_ = db.Create(&roleVP).Error

	gradUser := model.User{
		ID:           202,
		StudentID:    "202",
		Username:     "grad_vp",
		Name:         "毕业老会长",
		DepartmentID: dept.ID,
		RoleID:       roleVP.ID,
		IsGraduated:  true,
	}
	_ = db.Create(&gradUser).Error

	r := gin.New()
	r.POST("/user/batch", func(c *gin.Context) {
		c.Set("user_id", gradUser.ID)
		h.BatchUserInfo(c)
	})

	body := strings.NewReader(`[{"user_id":999,"department_id":4,"role_id":12}]`)
	req, _ := http.NewRequest(http.MethodPost, "/user/batch", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for graduated user calling batch, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_BatchUserInfo_SuccessForViceMinister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, db := setupTestUserHandler(t)

	dept := model.Department{ID: 5, Name: "技术部", Type: "部门", IsActive: true}
	_ = db.Create(&dept).Error
	roleViceMin := model.Role{ID: 20, Name: "副部长", Level: 50, Type: "部门", IsActive: true}
	_ = db.Create(&roleViceMin).Error
	roleClerk := model.Role{ID: 21, Name: "干事", Level: 20, Type: "部门", IsActive: true}
	_ = db.Create(&roleClerk).Error
	roleVP := model.Role{ID: 22, Name: "副会长", Level: 80, Type: "理事会", IsActive: true}
	_ = db.Create(&roleVP).Error

	vpMin := model.User{
		ID:           301,
		StudentID:    "301",
		Username:     "vp_min",
		Name:         "副部长",
		DepartmentID: dept.ID,
		RoleID:       roleViceMin.ID,
		IsGraduated:  false,
	}
	targetUser := model.User{
		ID:           302,
		StudentID:    "302",
		Username:     "target_user",
		Name:         "目标干事",
		DepartmentID: dept.ID,
		RoleID:       roleClerk.ID,
		IsGraduated:  false,
	}
	_ = db.Create(&vpMin).Error
	_ = db.Create(&targetUser).Error

	r := gin.New()
	r.POST("/user/batch", func(c *gin.Context) {
		c.Set("user_id", vpMin.ID)
		h.BatchUserInfo(c)
	})

	body := strings.NewReader(`[{"user_id":302,"department_id":5,"role_id":21}]`)
	req, _ := http.NewRequest(http.MethodPost, "/user/batch", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for vice minister calling batch, got %d, body: %s", w.Code, w.Body.String())
	}
}
