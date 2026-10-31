package handler

import (
	"net/http"
	"net/http/httptest"
	"suseoaa/internal/request"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseQueryBool(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		url         string
		key         string
		expectedVal *bool
		expectErr   bool
	}{
		{"/test", "has_result", nil, false},
		{"/test?has_result=", "has_result", nil, false},
		{"/test?has_result=   ", "has_result", nil, false},
		{"/test?has_result=null", "has_result", nil, false},
		{"/test?has_result=undefined", "has_result", nil, false},
		{"/test?has_result=all", "has_result", nil, false},
		{"/test?has_result=false", "has_result", func() *bool { b := false; return &b }(), false},
		{"/test?has_result=true", "has_result", func() *bool { b := true; return &b }(), false},
		{"/test?has_result=0", "has_result", func() *bool { b := false; return &b }(), false},
		{"/test?has_result=1", "has_result", func() *bool { b := true; return &b }(), false},
		{"/test?has_result=invalid", "has_result", nil, true},
	}

	for _, tc := range testCases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, tc.url, nil)

		val, err := parseQueryBool(c, tc.key)
		if tc.expectErr {
			if err == nil {
				t.Errorf("url %s: expected error, got nil", tc.url)
			}
			continue
		}
		if err != nil {
			t.Errorf("url %s: unexpected error: %v", tc.url, err)
			continue
		}

		if tc.expectedVal == nil {
			if val != nil {
				t.Errorf("url %s: expected nil, got %v", tc.url, *val)
			}
		} else {
			if val == nil {
				t.Errorf("url %s: expected %v, got nil", tc.url, *tc.expectedVal)
			} else if *val != *tc.expectedVal {
				t.Errorf("url %s: expected %v, got %v", tc.url, *tc.expectedVal, *val)
			}
		}
	}
}

func TestGetApplicationListReq_BindEmptyQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 模拟 Apifox 中勾选了参数名但未填值时的 URL：?department_id=&term_id=7&has_result=
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/v2/application/list?department_id=&term_id=7&has_result=", nil)

	var req request.GetApplicationListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		t.Fatalf("unexpected bind err: %v", err)
	}

	hasResult, err := parseQueryBool(c, "has_result")
	if err != nil {
		t.Fatalf("unexpected parse err: %v", err)
	}
	req.HasResult = hasResult

	if req.HasResult != nil {
		t.Fatalf("expected HasResult to be nil (all applications), but got %v", *req.HasResult)
	}
	if req.TermID != 7 {
		t.Fatalf("expected TermID 7, got %d", req.TermID)
	}
}
