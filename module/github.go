package module

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var (
	githubOauthConfig *oauth2.Config
	transformer_url   string
)

var respData struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	transformer_url = os.Getenv("TRANSFORMER_URL")
	githubOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		RedirectURL:  os.Getenv("REDIRECT_URL"),
		Scopes:       []string{"read:user", "user:email"}, // 请求用户信息和邮箱权限
		Endpoint:     github.Endpoint,
	}
}

func RequestGithubUserInfo(c *gin.Context, code string) (map[string]interface{}, error) {
	// 使用OAuth配置对象中定义的Exchange方法，通过code获取access token
	token, err := githubOauthConfig.Exchange(c, code)
	if err != nil {
		logger.Error("failed to exchange token:", zap.Error(err))
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("获取token错误"))
		return nil, err
	}
	client := githubOauthConfig.Client(c, token)
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

func GithubLogin(c *gin.Context, code string) {
	defer func() {
		if err := recover(); err != nil {
			if e, ok := err.(error); ok {
				logger.Error("failed to github login:", zap.Error(e))
				c.AbortWithError(http.StatusBadRequest, fmt.Errorf("github登录错误"))
			} else {
				logger.Error("failed to github login:", zap.Any("error", err))
				c.AbortWithError(http.StatusBadRequest, fmt.Errorf("github登录错误"))
			}
		}
	}()
	userInfo, err := RequestGithubUserInfo(c, code)
	if err != nil {
		logger.Error("failed to get user info:", zap.Error(err))
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("获取github用户信息错误"))
		return
	}
	github := userInfo["login"].(string)
	// 构造请求 URL
	reqURL, err := url.Parse(transformer_url + "/api/users/thirdPartyLogin")
	if err != nil {
		// 处理 URL 解析错误
		fmt.Printf("Error parsing URL: %v\n", err)
		return
	}
	// 构建请求体数据
	requestBody := map[string]string{
		"third_party_type": "github",
		"third_party_id":   github,
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		// 处理请求体数据序列化错误
		fmt.Printf("Error serializing request body: %v\n", err)
		return
	}

	// 发送 POST 请求
	resp, err := http.Post(reqURL.String(), "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		// 处理请求错误
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		// 处理响应体解析错误
		fmt.Printf("Error parsing response body: %v\n", err)
		return
	}
	// 输出响应数据中的 JWT 字段
	fmt.Printf("JWT: %s\n", respData.Token)
	// 返回响应数据为 json, {github,jwt:respData.JWT}
	c.JSON(http.StatusOK, gin.H{"github": github, "jwt": respData.Token})
}
