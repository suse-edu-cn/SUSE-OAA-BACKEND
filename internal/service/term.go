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
	return t.TermRepo.CreateApplication(application)
}
