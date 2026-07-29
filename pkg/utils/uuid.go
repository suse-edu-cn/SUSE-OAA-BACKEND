package utils

import (
	"errors"

	"github.com/google/uuid"
)

func GetUUID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", errors.New("生成uuid失败 " + err.Error())
	}
	return id.String(), nil
}
