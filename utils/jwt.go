package utils

import (
	"fmt"
	"os"

	"github.com/dgrijalva/jwt-go"
)

type RequestBody struct {
	JWT          string `json:"jwt"`
	Code         string `json:"code"`
	PlatformType string `json:"type"`
}

type RequestLoginBody struct {
	Code         string `json:"code"`
	PlatformType string `json:"type"`
}

func JwtDecode(jwtToken string) (string, error) {
	// 解析JWT
	parsedToken, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		// 验证算法和密钥
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		return "", err
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return "", err
	}
	address := claims["address"].(string)
	return address, nil
}
