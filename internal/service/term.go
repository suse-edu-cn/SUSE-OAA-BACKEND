package service

import (
	"errors"
	"suseoaa/internal/model"
	"suseoaa/internal/repository"
)

type TermService struct {
	TermRepo    repository.TermRepository
	UserService UserService
}

func NewTerService(termRepo repository.TermRepository, service UserService) TermService {
	return TermService{
		TermRepo:    termRepo,
		UserService: service,
	}
}
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
	err = t.TermRepo.UpdateTerm(term)
	if err != nil {
		return err
	}
	return nil
}
