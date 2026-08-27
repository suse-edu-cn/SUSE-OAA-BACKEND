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

func (t *TermService) CreateTerm(userID uint64, term model.Term) error {
	err := term.CheckPeriod()
	if err != nil {
		return err
	}
	level, _, err := t.UserService.Repo.GetRoleLevelAndDepartment(userID)
	if err != nil {
		return err
	}
	if level < 80 {
		return errors.New("权力不够")
	}
	err = t.TermRepo.CreateTerm(term)
	if err != nil {
		return err
	}
	return nil
}
