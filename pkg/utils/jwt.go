package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type MyClaims struct {
	Name      string `json:"name"`
	UserID    uint64 `json:"userid"`
	StudentID string `json:"student_id"`
	jwt.RegisteredClaims
}

func GenerateToken(name string, userID uint64, studentID string, secret string, expireHour int) (string, error) {
	claims := MyClaims{
		Name:      name,
		UserID:    userID,
		StudentID: studentID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHour) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "GrowthOS",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func ParseToken(tokenString string, secret string) (*MyClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*MyClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("token类型不合法")
}
