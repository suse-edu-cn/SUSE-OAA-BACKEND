package request

import "testing"

func TestRoleCheckType(t *testing.T) {
	validTypes := []string{"部门", "协会", "理事会"}
	for _, vt := range validTypes {
		c := CreateRoleReq{Type: vt}
		if err := c.CheckType(); err != nil {
			t.Errorf("CreateRoleReq.CheckType() failed for valid type %s: %v", vt, err)
		}
		u := UpdateRoleReq{Type: vt}
		if err := u.CheckType(); err != nil {
			t.Errorf("UpdateRoleReq.CheckType() failed for valid type %s: %v", vt, err)
		}
	}

	invalidTypes := []string{"", "其他", "学校", "委员会"}
	for _, it := range invalidTypes {
		c := CreateRoleReq{Type: it}
		if err := c.CheckType(); err == nil {
			t.Errorf("CreateRoleReq.CheckType() should fail for invalid type %s", it)
		}
		u := UpdateRoleReq{Type: it}
		if err := u.CheckType(); err == nil {
			t.Errorf("UpdateRoleReq.CheckType() should fail for invalid type %s", it)
		}
	}
}

func TestDepartmentCheckType(t *testing.T) {
	validTypes := []string{"部门", "协会", "理事会"}
	for _, vt := range validTypes {
		c := CreateDepartmentReq{Type: vt}
		if err := c.CheckType(); err != nil {
			t.Errorf("CreateDepartmentReq.CheckType() failed for valid type %s: %v", vt, err)
		}
		u := UpdateDepartmentReq{Type: vt}
		if err := u.CheckType(); err != nil {
			t.Errorf("UpdateDepartmentReq.CheckType() failed for valid type %s: %v", vt, err)
		}
	}

	invalidTypes := []string{"", "未知", "院系"}
	for _, it := range invalidTypes {
		c := CreateDepartmentReq{Type: it}
		if err := c.CheckType(); err == nil {
			t.Errorf("CreateDepartmentReq.CheckType() should fail for invalid type %s", it)
		}
		u := UpdateDepartmentReq{Type: it}
		if err := u.CheckType(); err == nil {
			t.Errorf("UpdateDepartmentReq.CheckType() should fail for invalid type %s", it)
		}
	}
}
