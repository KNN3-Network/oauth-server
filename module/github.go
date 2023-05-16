package module

import (
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
	JWT string `json:"jwt"`
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

func RedirectGithubLogin(c *gin.Context, code string) {
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
	// 构建查询参数
	queryParams := url.Values{}
	queryParams.Add("type", "github")
	queryParams.Add("id", github)
	reqURL.RawQuery = queryParams.Encode()

	// 发送 GET 请求
	resp, err := http.Get(reqURL.String())
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
	fmt.Printf("JWT: %s\n", respData.JWT)
	// 设置 Cookie
	cookie := &http.Cookie{
		Name:  "jwt",
		Value: respData.JWT,
		Path:  "/",
	}
	http.SetCookie(c.Writer, cookie)

	// 重定向到目标 URL
	c.Redirect(http.StatusFound, transformer_url)
}
