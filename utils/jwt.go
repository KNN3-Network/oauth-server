package utils

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"
)

func JwtDecode(jwtToken string) (string, error) {
	// 解析JWT
	parsedToken, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		// 验证算法和密钥
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}
		return []byte("my_secret_key"), nil
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
