package request

import (
	"encoding/json"
	"time"
)

// -----------------------------------------------------------
// 招新换届
var shanghaiLocation, _ = time.LoadLocation("Asia/Shanghai")

type CreateTermReq struct {
	Year        uint64     `json:"year" binding:"required"`
	Type        string     `json:"type" binding:"required"`
	Title       string     `json:"title" binding:"required"`
	EditPeriod  TimePeriod `json:"edit_period" binding:"required"`
	QueryPeriod TimePeriod `json:"query_period" binding:"required"`
}
type UpdateTermReq struct {
	ID          uint64     `json:"id" binding:"required"`
	Title       string     `json:"title" binding:"required"`
	EditPeriod  TimePeriod `json:"edit_period" binding:"required"`
	QueryPeriod TimePeriod `json:"query_period" binding:"required"`
}

type TimePeriod struct {
	StartAt time.Time `json:"start_at" binding:"required"`
	EndAt   time.Time `json:"end_at" binding:"required"`
}

func (t *TimePeriod) UnmarshalJSON(data []byte) error {
	type Alias struct {
		StartAt string `json:"start_at"`
		EndAt   string `json:"end_at"`
	}

	var v Alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	start, err := time.ParseInLocation("2006-01-02", v.StartAt, shanghaiLocation)
	if err != nil {
		return err
	}

	end, err := time.ParseInLocation("2006-01-02", v.EndAt, shanghaiLocation)
	if err != nil {
		return err
	}

	t.StartAt = start
	t.EndAt = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	return nil
}

type GetTermListReq struct {
	Year uint64 `form:"year"`
	Type string `form:"type"`
}

// --------------------------------------------------
// 申请表

type OrganizationRole struct {
	DepartmentID uint64 `gorm:"default:0" json:"department_id"` // 部门ID
	RoleID       uint64 `gorm:"default:0" json:"role_id"`       // 职位/角色ID
}
type CreateApplicationReq struct {
	TermID          uint64           `json:"term_id" binding:"required"`
	College         string           `json:"college" binding:"required"`
	MajorClass      string           `json:"major_class" binding:"required"`
	Gender          string           `json:"gender" binding:"required"`
	Phone           string           `json:"phone" binding:"required"`
	QQ              string           `json:"qq" binding:"required"`
	PoliticalStatus string           `json:"political_status" binding:"required"`
	BirthDate       string           `json:"birth_date" binding:"required"`
	FirstChoice     OrganizationRole `json:"first_choice" binding:"required"`
	SecondChoice    OrganizationRole `json:"second_choice" binding:"required"`
	AllowAdjust     *bool            `json:"allow_adjust" binding:"required"`
	Resume          string           `json:"resume" binding:"required"`
	Reason          string           `json:"reason" binding:"required"`
}
type GetApplicationDepartmentReq struct {
	RoleID uint64 `form:"role_id"`
}
type GetApplicationRoleReq struct {
	DepartmentID uint64 `form:"department_id"`
}

type UpdateApplicationReq struct {
	College         string           `json:"college" binding:"required"`
	MajorClass      string           `json:"major_class" binding:"required"`
	Gender          string           `json:"gender" binding:"required"`
	Phone           string           `json:"phone" binding:"required"`
	QQ              string           `json:"qq" binding:"required"`
	PoliticalStatus string           `json:"political_status" binding:"required"`
	BirthDate       string           `json:"birth_date" binding:"required"`
	FirstChoice     OrganizationRole `json:"first_choice" binding:"required"`
	SecondChoice    OrganizationRole `json:"second_choice" binding:"required"`
	AllowAdjust     *bool            `json:"allow_adjust" binding:"required"`
	Resume          string           `json:"resume" binding:"required"`
	Reason          string           `json:"reason" binding:"required"`
}

//------------------------------
//面试官

type CreateInterviewer struct {
	TermID       uint64        `json:"term_id" binding:"required"`
	Interviewers []Interviewer `json:"interviewers" binding:"required"`
}
type Interviewer struct {
	UserID uint64 `json:"user_id" binding:"required"`
	Remark string `json:"remark" binding:"required"`
}

type GetInterviewerListReq struct {
	TermID uint64 `form:"term_id"`
}
