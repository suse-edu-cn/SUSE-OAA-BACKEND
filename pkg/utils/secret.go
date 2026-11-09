package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

func GenerateClientSecret() (plainSecret string, maskSecret string, err error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	randomHex := hex.EncodeToString(bytes)
	plainSecret = "oaa_sec_" + randomHex
	prefix := "oaa_sec_" + randomHex[:4]
	suffix := randomHex[len(randomHex)-4:]
	maskSecret = fmt.Sprintf("%s%s%s", prefix, strings.Repeat("*", 20), suffix)
	return plainSecret, maskSecret, nil
}
