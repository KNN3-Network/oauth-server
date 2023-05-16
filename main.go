package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/KNN3-Network/oauth-server/module"
	"github.com/KNN3-Network/oauth-server/utils"
	"github.com/gin-contrib/cors"
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

var stackoverflow = new(module.Stackoverflow)

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
	r.Use(cors.Default())

	r.POST("/oauth/bind", func(c *gin.Context) {
		var requestBody utils.RequestBody
		// 将请求体中的 JSON 数据绑定到结构体
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			// 处理绑定错误
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		jwt := requestBody.JWT
		code := requestBody.Code
		platformType := requestBody.PlatformType
		if jwt == "" || code == "" || platformType == "" {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("参数错误"))
			return
		}
		db := utils.GetDB()
		address, err := utils.JwtDecode(jwt)
		logger.Info("JwtDecode address", zap.Any("address", address))

		if err != nil {
			logger.Error("failed to decode jwt:", zap.Error(err))
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("解析jwt错误"))
			return
		}
		if platformType == "github" {
			// 使用OAuth配置对象中定义的Exchange方法，通过code获取access token
			token, err := githubOauthConfig.Exchange(c, code)
			if err != nil {
				logger.Error("failed to exchange token:", zap.Error(err))
				c.AbortWithError(http.StatusBadRequest, fmt.Errorf("获取token错误"))
				return
			}
			client := githubOauthConfig.Client(c, token)
			userInfo, err := getUserInfo(client)
			if err != nil {
				logger.Error("failed to get user info:", zap.Error(err))
				c.AbortWithError(http.StatusBadRequest, fmt.Errorf("获取github用户信息错误"))
				return
			}
			github := userInfo["login"].(string)
			bind := utils.OauthBind{}
			result := db.Model(&utils.OauthBind{}).Where("github = ?", github).First(&bind)
			// 判断返回结果里面github是不是空
			if bind != (utils.OauthBind{}) {
				logger.Error("github has bound:", zap.Error(result.Error))
				c.JSON(http.StatusOK, gin.H{"data": "false"})
				return
			}
			logger.Info("userInfo", zap.Any("user", userInfo))
			bind = utils.OauthBind{}
			result = db.Model(&utils.OauthBind{}).Where("addr = ?", address).First(&bind)
			// 判断返回结果里面address是不是空
			if bind != (utils.OauthBind{}) {
				result = db.Model(&bind).Where("addr = ?", address).Updates(map[string]interface{}{"github": github})
				if result.Error != nil {
					logger.Error("failed to update address:", zap.Error(result.Error))
					c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Update Error"))
					return
				}
			} else {
				// insert into oauth_binds (addr, github) values (address, github)
				result = db.Model(&utils.OauthBind{}).Create(&utils.OauthBind{Addr: address, Github: github})
				if result.Error != nil {
					logger.Error("failed to insert oauth_bind:", zap.Error(result.Error))
					c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Insert Error"))
					return
				}
			}

			c.JSON(http.StatusOK, gin.H{"data": "success"})
		} else if platformType == "discord" {
			token, err := utils.ExchangeCodeForToken(code)
			if err != nil {
				logger.Error("failed to exchange discord token:", zap.Error(err))
				c.AbortWithError(http.StatusBadRequest, fmt.Errorf("获取token错误"))
				return
			}
			user, err := utils.FetchUser(token)
			if err != nil {
				logger.Error("failed to get discord user info:", zap.Error(err))
				c.AbortWithError(http.StatusBadRequest, fmt.Errorf("获取discord用户信息错误"))
				return
			}
			addr := utils.OauthBind{}
			result := db.Model(&utils.OauthBind{}).Where("discord = ?", user.ID).First(&addr)
			if addr != (utils.OauthBind{}) {
				logger.Error("discord has bound:", zap.Error(result.Error))
				c.JSON(http.StatusOK, gin.H{"data": "false"})
				return
			}
			logger.Info("userInfo", zap.Any("user", user.ID))
			bind := utils.OauthBind{}
			result = db.Model(&utils.OauthBind{}).Where("addr = ?", address).First(&bind)
			// 判断返回结果里面address是不是空
			if bind != (utils.OauthBind{}) {
				result = db.Model(&bind).Where("addr = ?", address).Updates(map[string]interface{}{"discord": user.ID})
				if result.Error != nil {
					logger.Error("failed to update address:", zap.Error(result.Error))
					c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Update Error"))
					return
				}
			} else {
				result = db.Model(&utils.OauthBind{}).Create(&utils.OauthBind{Addr: address, Discord: user.ID})
				if result.Error != nil {
					logger.Error("failed to insert oauth_bind:", zap.Error(result.Error))
					c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Insert Error"))
					return
				}
			}
			c.JSON(http.StatusOK, gin.H{"data": "success"})
		}

		switch platformType {
		case "stackexchange":
			stackoverflow.Bind(c, code, address)
		default:
			logger.Info("error platformType", zap.String("platformType", platformType))
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

		c.Redirect(http.StatusTemporaryRedirect, "https://topscore.social/pass/succss?type=github&code="+code)
	})

	// github oauth
	r.GET("/oauth/discord", func(c *gin.Context) {
		code := c.Query("code")
		if code == "" {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("No authorization code provided."))
			return
		}
		logger.Info("discord oauth认证", zap.String("code", code))

		c.Redirect(http.StatusTemporaryRedirect, "https://topscore.social/pass/succss?type=discord&code="+code)
	})

	r.GET("/oauth/gmail", func(c *gin.Context) {
		code := c.Query("code")
		state := c.Query("state")
		if code == "" || state == "" {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("No authorization code provided."))
			return
		}
		logger.Info("discord oauth认证", zap.String("code", code))

		c.Redirect(http.StatusTemporaryRedirect, "https://topscore.social/pass/succss?type=gmail&code="+code)
	})

	// stackoverflow
	v1 := r.Group("/oauth/stackoverflow")
	{
		// stackoverflow := new(module.Stackoverflow)
		v1.GET("/authcodeurl", stackoverflow.AuthCodeURL)
		v1.GET("/", stackoverflow.CallBack)

	}

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
