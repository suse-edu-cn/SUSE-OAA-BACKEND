package model

import (
	"errors"
	"time"

	"gorm.io/plugin/soft_delete"
)

type Term struct {
	ID    uint64 `gorm:"primaryKey;autoIncrement" json:"id"` // 周期唯一自增ID
	Year  uint64 `gorm:"not null;uniqueIndex:idx_year_type" json:"year"`
	Type  string `gorm:"size:16;not null;uniqueIndex:idx_year_type" json:"type"`
	Title string `gorm:"size:64;not null" json:"title"` // 周期展示标题 (如: "2026年秋季招新")

	// 修改/填报时间窗口 (Edit: 编辑/修改)
	EditStartAt time.Time `gorm:"not null" json:"edit_start_at"` // 允许填报/修改的开始时间
	EditEndAt   time.Time `gorm:"not null" json:"edit_end_at"`   // 允许填报/修改的截止时间 (截止后表单锁定)

	// 结果公示/查询时间窗口 (Query: 查询)
	QueryStartAt time.Time `gorm:"not null" json:"query_start_at"` // 结果公示/允许查询的开始时间
	QueryEndAt   time.Time `gorm:"not null" json:"query_end_at"`   // 结果公示/允许查询的结束时间

	// 最终批量执行状态 (Execute: 执行/生效)
	IsExecuted     bool       `gorm:"not null;default:false;index:idx_term_execution" json:"is_executed"` // 查询期结束后，是否已执行批量更新 users 表的事务
	ExecuteAfterAt time.Time  `gorm:"not null;index:idx_term_execution" json:"execute_after_at"`
	ExecutedAt     *time.Time `json:"executed_at"` // 事务具体执行生效的时间戳

	CreatedAt time.Time             `json:"created_at"`                                          // 记录创建时间
	UpdatedAt time.Time             `json:"updated_at"`                                          // 记录更新时间
	DeletedAt soft_delete.DeletedAt `gorm:"softDelete:milli;uniqueIndex:idx_year_type" json:"-"` // GORM 软删除标记
}

func (Term) TableName() string { return "terms" }
func (t Term) CheckPeriod() error {
	if t.EditStartAt.IsZero() ||
		t.EditEndAt.IsZero() ||
		t.QueryStartAt.IsZero() ||
		t.QueryEndAt.IsZero() {
		return errors.New("部分时间不存在")
	}

	if t.EditStartAt.Before(t.EditEndAt) &&
		t.EditEndAt.Before(t.QueryStartAt) &&
		t.QueryStartAt.Before(t.QueryEndAt) {
		return nil
	}

	return errors.New("时间不规范")
}
func (t Term) IsInEditPeriod(now time.Time) bool {
	return !now.Before(t.EditStartAt) && !now.After(t.EditEndAt)
}

func (t Term) IsInQueryPeriod(now time.Time) bool {
	return !now.Before(t.QueryStartAt) && !now.After(t.QueryEndAt)
}

type Interviewer struct {
	ID           uint64                `gorm:"primaryKey;autoIncrement" json:"id"`
	TermID       uint64                `gorm:"uniqueIndex:idx_term_user;not null" json:"term_id"`
	UserID       uint64                `gorm:"uniqueIndex:idx_term_user;not null" json:"user_id"`
	DepartmentID uint64                `gorm:"not null" json:"department_id"`
	Remark       string                `gorm:"size:255" json:"remark"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
	DeletedAt    soft_delete.DeletedAt `gorm:"softDelete:milli;uniqueIndex:idx_term_user" json:"-"`
}
type InterviewerInfo struct {
	ID             uint64 `json:"id"`
	Year           uint64 `json:"year"`
	TermType       string `json:"term_type"`
	Name           string `json:"name"`
	Role           string `json:"role"`
	DepartmentName string `json:"department_name"`
	Remark         string `json:"remark"`
}

func (Interviewer) TableName() string { return "interviewers" }

const (
	TermTypeRecruit  string = "招新"
	TermTypeElection string = "换届"
)

const (
	DecisionPending          string = "待定"
	DecisionAdmittedFirst    string = "录取第一志愿"
	DecisionAdmittedSecond   string = "录取第二志愿"
	DecisionAdmittedAdjusted string = "已调剂"
	DecisionRejected         string = "未通过"
)

type OrganizationRole struct {
	DepartmentID uint64 `gorm:"default:0;not null" json:"department_id"` // 部门ID
	RoleID       uint64 `gorm:"default:0;not null" json:"role_id"`       // 职位/角色ID
}

type Application struct {
	ID              uint64           `gorm:"primaryKey;autoIncrement" json:"id"`                      // 申请单自增ID
	TermID          uint64           `gorm:"not null;uniqueIndex:idx_user_term" json:"term_id"`       // 关联的周期ID (对应 terms.id)
	Type            string           `gorm:"size:16;not null;index" json:"type"`                      // 业务类型: 招新 或 换届
	UserID          uint64           `gorm:"index;not null;uniqueIndex:idx_user_term" json:"user_id"` // 申请人系统账号ID (对应 users.id)
	Name            string           `gorm:"size:32;not null" json:"name"`                            // 申请人姓名
	Gender          string           `gorm:"size:8;not null" json:"gender"`                           // 性别 (男 / 女)
	Avatar          string           `gorm:"size:255" json:"avatar"`                                  // 照片/证件照链接
	StudentID       string           `gorm:"size:32;index;not null" json:"student_id"`                // 学号
	College         string           `gorm:"size:64;not null" json:"college"`                         // 所在学院 (如: 计算机科学与工程学院)
	MajorClass      string           `gorm:"size:64;not null" json:"major_class"`                     // 专业班级 (如: 计科241)
	PoliticalStatus string           `gorm:"size:32;not null" json:"political_status"`                // 政治面貌 (如: 共青团员 / 中共党员)
	BirthDate       string           `gorm:"size:32;not null" json:"birth_date"`                      // 出生年月 (如: "2005-09")
	QQ              string           `gorm:"size:20;not null" json:"qq"`                              // QQ 号码
	Phone           string           `gorm:"size:20;not null" json:"phone"`                           // 联系电话 (手机号)
	FirstChoice     OrganizationRole `gorm:"embedded;embeddedPrefix:first_" json:"first_choice"`      // 第一志愿 (映射为 first_department_id, first_role_id)
	SecondChoice    OrganizationRole `gorm:"embedded;embeddedPrefix:second_" json:"second_choice"`    // 第二志愿 (映射为 second_department_id, second_role_id)
	AllowAdjust     *bool            `gorm:"default:true" json:"allow_adjust"`                        // 是否服从调剂

	Resume string `gorm:"type:text" json:"resume"` // 个人简历 / 在会工作经历
	Reason string `gorm:"type:text" json:"reason"` // 竞选理由 / 申请阐述

	Decision       string           `gorm:"size:32;default:'待定';not null;index" json:"decision"` // 录取决策状态 (默认: '待定')
	Result         OrganizationRole `gorm:"embedded;embeddedPrefix:result_" json:"result"`       // 最终录取结果 (映射为 result_department_id, result_role_id)
	OperatorID     uint64           `gorm:"default:0" json:"operator_id"`                        // 做出决定的操作人UserID
	DecisionRemark string           `gorm:"size:255" json:"decision_remark"`                     // 决策说明/调剂备注

	CreatedAt time.Time             `json:"created_at"`                                          // 提交时间
	UpdatedAt time.Time             `json:"updated_at"`                                          // 修改时间
	DeletedAt soft_delete.DeletedAt `gorm:"softDelete:milli;uniqueIndex:idx_user_term" json:"-"` // 软删除标记
}

func (Application) TableName() string { return "applications" }

type InterviewResult struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	TermID        uint64 `gorm:"not null;index" json:"term_id"`
	ApplicationID uint64 `gorm:"not null;uniqueIndex:idx_app_del" json:"application_id"`
	Type          string `gorm:"size:16;not null;index" json:"type"`
	UserID        uint64 `gorm:"not null;index" json:"user_id"`

	Decision string `gorm:"size:32;not null;index" json:"decision"`

	ResultDepartmentID uint64 `gorm:"not null" json:"result_department_id"`
	ResultRoleID       uint64 `gorm:"not null" json:"result_role_id"`

	// 执行前保存用户原始组织信息；未执行时为 0。
	Old        OrganizationRole `gorm:"embedded;embeddedPrefix:old_" json:"old"`
	ExecutedAt *time.Time       `json:"executed_at"`

	OperatorID uint64 `gorm:"not null" json:"operator_id"`
	Remark     string `gorm:"size:255" json:"remark"`

	DeletedAt soft_delete.DeletedAt `gorm:"softDelete:milli;not null;default:0;uniqueIndex:idx_app_del" json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (InterviewResult) TableName() string {
	return "interview_results"
}
