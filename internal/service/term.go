package service

import (
	"errors"
	"suseoaa/internal/model"
	"suseoaa/internal/repository"
	"time"
)

type TermService struct {
	TermRepo    repository.TermRepository
	UserService UserService
}

func NewTermService(termRepo repository.TermRepository, service UserService) TermService {
	return TermService{
		TermRepo:    termRepo,
		UserService: service,
	}
}

//-----------------------
//业务周期

func (t *TermService) CheckLevel(userID uint64) error {
	level, _, err := t.UserService.Repo.GetRoleLevelAndDepartment(userID)
	if err != nil {
		return err
	}
	if level < 80 {
		return errors.New("权力不够")
	}
	return nil
}
func (t *TermService) CreateTerm(term model.Term) error {
	err := term.CheckPeriod()
	if err != nil {
		return err
	}
	term.ExecutedAt = term.QueryEndAt.Add(1 * time.Minute)
	err = t.TermRepo.CreateTerm(term)
	if err != nil {
		return err
	}
	return nil
}
func (t *TermService) UpdateTerm(term model.Term) error {
	err := term.CheckPeriod()
	if err != nil {
		return err
	}
	term.ExecutedAt = term.QueryEndAt.Add(1 * time.Minute)
	err = t.TermRepo.UpdateTerm(term)
	if err != nil {
		return err
	}
	return nil
}
func (t *TermService) GetTermList(year uint64, termType string) ([]*model.Term, error) {
	termList, err := t.TermRepo.GetTermList(year, termType)
	if err != nil {
		return nil, err
	}
	if len(termList) == 0 {
		return nil, errors.New("无匹配数据")
	}
	return termList, nil
}

//----------------------------
//申请表

func (t *TermService) CreateApplication(application model.Application) error {
	err := t.CheckApplicationDepartmentPosition(application)
	if err != nil {
		return err
	}
	term, err := t.TermRepo.GetTermByID(application.TermID)
	if err != nil {
		return err
	}
	application.Type = term.Type
	return t.TermRepo.CreateApplication(application)
}

func (t *TermService) CheckApplicationDepartmentPosition(application model.Application) error {
	err := t.UserService.VerifyDepartmentPosition(application.FirstChoice.DepartmentID, application.FirstChoice.RoleID)
	if err != nil {
		return err
	}
	err = t.UserService.VerifyDepartmentPosition(application.SecondChoice.DepartmentID, application.SecondChoice.RoleID)
	if err != nil {
		return err
	}
	level, err := t.UserService.RoleRepo.GetLevelByID(application.FirstChoice.RoleID)
	if err != nil {
		return err
	}
	if level > 90 {
		return errors.New("不能申请成为该职位")
	}
	return nil
}
func (t *TermService) GetRolesByDepartmentsID(departmentID uint64) ([]*model.Role, error) {
	var departmentType string
	var err error
	departmentType = ""
	if departmentID != 0 {
		departmentType, err = t.UserService.DepartmentRepo.GetTypeByDepartmentID(departmentID)
		if err != nil {
			return nil, err
		}
	}
	roles, err := t.UserService.RoleRepo.GetRoleByType(departmentType)
	if err != nil {
		return nil, err
	}
	var result []*model.Role
	for _, role := range roles {
		if role.Level <= 90 {
			result = append(result, role)
		}
	}
	return result, nil
}
func (t *TermService) GetDepartmentByRoleID(roleID uint64) ([]*model.Department, error) {
	var roleType string
	var err error
	roleType = ""
	if roleID != 0 {
		roleType, err = t.UserService.RoleRepo.GetTypeByRoleID(roleID)
		if err != nil {
			return nil, err
		}
	}
	departments, err := t.UserService.DepartmentRepo.GetDepartmentByType(roleType)
	if err != nil {
		return nil, err
	}
	return departments, nil
}
