package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"maps"
	"math/big"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWK struct {
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}
type OIDCManager struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	KeyID      string
	Issuer     string
}

func NewOIDCManager(privateKeyPath, issuer, keyID string) (*OIDCManager, error) {
	var privateKey *rsa.PrivateKey
	if data, err := os.ReadFile(privateKeyPath); err == nil {
		block, _ := pem.Decode(data)
		if block == nil {
			return nil, errors.New("解析私钥 PEM 失败")
		}
		var parseErr error
		privateKey, parseErr = x509.ParsePKCS1PrivateKey(block.Bytes)
		if parseErr != nil {
			k, err8 := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err8 != nil {
				return nil, parseErr
			}
			var ok bool
			privateKey, ok = k.(*rsa.PrivateKey)
			if !ok {
				return nil, errors.New("私钥类型不是 RSA")
			}
		}
	} else {
		var genErr error
		privateKey, genErr = rsa.GenerateKey(rand.Reader, 2048)
		if genErr != nil {
			return nil, genErr
		}
		der := x509.MarshalPKCS1PrivateKey(privateKey)
		pemBytes := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: der,
		})
		_ = os.WriteFile(privateKeyPath, pemBytes, 0600)
	}
	return &OIDCManager{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		KeyID:      keyID,
		Issuer:     issuer,
	}, nil

}

func (m *OIDCManager) GetJWKS() JWKS {
	nBytes := m.PublicKey.N.Bytes()
	eBytes := big.NewInt(int64(m.PublicKey.E)).Bytes()
	return JWKS{
		Keys: []JWK{
			{
				Kty: "RSA",
				Use: "sig",
				Alg: "RS256",
				Kid: m.KeyID,
				N:   base64.RawURLEncoding.EncodeToString(nBytes),
				E:   base64.RawURLEncoding.EncodeToString(eBytes),
			},
		},
	}
}

func (m *OIDCManager) GenerateIDToken(sub, aud string, expireDuration time.Duration, extraClaims map[string]any) (string, error) {
	claims := jwt.MapClaims{
		"iss": m.Issuer,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(expireDuration).Unix(),
		"sub": sub,
		"aud": aud,
	}
	maps.Copy(claims, extraClaims)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = m.KeyID
	return token.SignedString(m.PrivateKey)
}
