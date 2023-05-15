package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/KNN3-Network/oauth-server/utils"
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

	r.POST("/oauth/bind", func(c *gin.Context) {
		jwt := c.PostForm("jwt")
		code := c.PostForm("code")
		platformType := c.PostForm("type")
		if jwt == "" || code == "" || platformType == "" {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("参数错误"))
			return
		}
		db := utils.GetDB()
		address, err := utils.JwtDecode(jwt)
		if err != nil {
			logger.Error("failed to decode jwt:", zap.Error(err))
			c.Redirect(http.StatusTemporaryRedirect, "/error")
			return
		}
		if platformType == "github" {
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
			github := userInfo["login"].(string)
			email, ok := userInfo["email"].(string)
			if !ok {
				email = ""
			}
			addr := utils.Address{}
			result := db.Model(&utils.Address{}).Where("github = ?", github).First(&addr)
			if addr != (utils.Address{}) {
				logger.Error("github has bound:", zap.Error(result.Error))
				c.Redirect(http.StatusTemporaryRedirect, "/error")
				return
			}
			logger.Info("userInfo", zap.Any("user", userInfo))
			addr = utils.Address{}

			result = db.Model(&addr).Where("addr = ?", address).Updates(map[string]interface{}{"github": github, "email": email})
			if result.Error != nil {
				logger.Error("failed to update address:", zap.Error(result.Error))
			}
			c.JSON(http.StatusOK, gin.H{"data": "success"})
		}
	})

	// github oauth
	r.GET("/oauth/github", func(c *gin.Context) {
		code := c.Query("code")
		if code == "" {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("No authorization code provided."))
			return
		}
		logger.Info("github oauth认证", zap.String("code", code))

		c.Redirect(http.StatusTemporaryRedirect, "https://topscore.social?code="+code)
	})

	// github oauth
	r.GET("/oauth/discord", func(c *gin.Context) {
		code := c.Query("code")
		if code == "" {
			logger.Error("No authorization code provided.")
			// todo 需要给个error页面
			c.Redirect(http.StatusTemporaryRedirect, "/error")
		}
		token, err := utils.ExchangeCodeForToken(code)
		if err != nil {
			logger.Error("Unable to exchange authorization code for access token", zap.Error(err))
			c.Redirect(http.StatusTemporaryRedirect, "/error")
			return
		}
		user, err := utils.FetchUser(token)
		if err != nil {
			logger.Error("Unable to fetch user information:", zap.Error(err))
			return
		}
		fmt.Println(user)
		// address, err := utils.JwtDecode(jwtToken)
		if err != nil {
			logger.Error("failed to get user info:", zap.Error(err))
			c.Redirect(http.StatusTemporaryRedirect, "/error")
			return
		}
		// fmt.Println(address)
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
