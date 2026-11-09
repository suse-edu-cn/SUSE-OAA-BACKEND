package utils

import (
	"errors"
	"unicode"
)

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("密码长度不能少于8位")
	}

	var hasLetter, hasDigit bool
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		} else if unicode.IsDigit(r) {
			hasDigit = true
		}
	}

	if !hasLetter || !hasDigit {
		return errors.New("密码必须包含字母和数字")
	}

	return nil
}
