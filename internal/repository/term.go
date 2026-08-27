package repository

import (
	"errors"
	"suseoaa/internal/model"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type TermRepository struct {
	DB *gorm.DB
}

func NewTermRepository(db *gorm.DB) TermRepository {
	return TermRepository{
		DB: db,
	}
}

//申请表

func (t *TermRepository) CreateApplication(application model.Application) error {
	return t.DB.Create(&application).Error
}

// 业务周期
func (t *TermRepository) CreateTerm(term model.Term) error {
	if err := t.DB.Create(&term).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return errors.New("创建失败，该年份类型的数据已存在")
		}
		return err
	}
	return nil
}

//面试官

//招新换届的历史
