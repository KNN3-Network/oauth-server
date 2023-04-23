package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/KNN3-Network/oauth-server/utils"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var logger = utils.Logger

var (
	githubOauthConfig *oauth2.Config
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	githubOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		RedirectURL:  os.Getenv("REDIRECT_URL"),
		Scopes:       []string{"read:user", "user:email"}, // 请求用户信息和邮箱权限
		Endpoint:     github.Endpoint,
	}
}

func main() {
	r := gin.Default()

	// github oauth
	r.GET("/oauth/github", func(c *gin.Context) {
		code := c.Query("code")
		jwtToken := c.Query("jwt")
		logger.Info("github oauth认证", zap.String("code", code))
		// 使用OAuth配置对象中定义的Exchange方法，通过code获取access token
		token, err := githubOauthConfig.Exchange(c, code)
		if err != nil {
			logger.Error("failed to exchange token:", zap.Error(err))
			// todo 需要给个error页面
			c.Redirect(http.StatusTemporaryRedirect, "/error")
			return
		}
		client := githubOauthConfig.Client(c, token)
		userInfo, err := getUserInfo(client)
		if err != nil {
			logger.Error("failed to get user info:", zap.Error(err))
			c.Redirect(http.StatusTemporaryRedirect, "/error")
			return
		}
		// 解析JWT
		parsedToken, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
			// 验证算法和密钥
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("invalid signing method")
			}
			return []byte("my_secret_key"), nil
		})

		if err != nil {
			logger.Error("解析token失败:", zap.Error(err))
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok || !parsedToken.Valid {
			logger.Error("token不符合规范")
		}
		db := utils.GetDB()
		address := claims["address"].(string)
		github := userInfo["login"].(string)
		email, ok := userInfo["email"].(string)
		if !ok {
			email = ""
		}

		logger.Info("userInfo", zap.Any("user", userInfo))
		addr := utils.Address{}

		result := db.Model(&addr).Where("addr = ?", address).Updates(map[string]interface{}{"github": github, "email": email})
		if result.Error != nil {
			logger.Error("failed to update address:", zap.Error(result.Error))
		}

		c.Redirect(http.StatusTemporaryRedirect, "https://topscore.social")
	})
	r.Run(":8001")
}

func getUserInfo(client *http.Client) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("get user info request failed: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get user info request failed: %w", err)
	}

	defer resp.Body.Close()

	var userInfo map[string]interface{}
	if err := decodeResponse(resp, &userInfo); err != nil {
		return nil, err
	}

	return userInfo, nil
}

// 辅助函数，用于从HTTP响应中反序列化JSON
func decodeResponse(resp *http.Response, v interface{}) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to decode response body: %w", err)
	}

	return nil
}
