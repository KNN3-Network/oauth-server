package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/KNN3-Network/oauth-server/module"
	"github.com/KNN3-Network/oauth-server/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var logger = utils.Logger

var stackoverflow = new(module.Stackoverflow)

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
			userInfo, err := module.RequestGithubUserInfo(c, code)
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
			token, err := module.ExchangeCodeForToken(code)
			if err != nil {
				logger.Error("failed to exchange discord token:", zap.Error(err))
				c.AbortWithError(http.StatusBadRequest, fmt.Errorf("获取token错误"))
				return
			}
			user, err := module.FetchUser(token)
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
			logger.Info("discord username", zap.Any("username", user.Username))
			logger.Info("discord avatar", zap.Any("user", user.Avatar))
			bind := utils.OauthBind{}
			result = db.Model(&utils.OauthBind{}).Where("addr = ?", address).First(&bind)
			// 判断返回结果里面address是不是空
			if bind != (utils.OauthBind{}) {
				result = db.Model(&bind).Where("addr = ?", address).Updates(map[string]interface{}{"discord": user.ID, "discord_name": user.Username})
				if result.Error != nil {
					logger.Error("failed to update address:", zap.Error(result.Error))
					c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Update Error"))
					return
				}
			} else {
				result = db.Model(&utils.OauthBind{}).Create(&utils.OauthBind{Addr: address, Discord: user.ID, DiscordName: user.Username})
				if result.Error != nil {
					logger.Error("failed to insert oauth_bind:", zap.Error(result.Error))
					c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Insert Error"))
					return
				}
			}
			c.JSON(http.StatusOK, gin.H{"data": "success"})
		} else if platformType == "stackexchange" {
			stackoverflow.Bind(c, code, address)
		} else if platformType == "gmail" {
			profile, err := module.GetGmailProfile(code)
			if err != nil {
				logger.Error("failed to exchange gmail token:", zap.Error(err))
				c.AbortWithError(http.StatusBadRequest, fmt.Errorf("获取token错误"))
				return
			}
			addr := utils.OauthBind{}
			result := db.Model(&utils.OauthBind{}).Where("gmail = ?", profile.EmailAddress).First(&addr)
			if addr != (utils.OauthBind{}) {
				logger.Error("gmail has bound:", zap.Error(result.Error))
				c.JSON(http.StatusOK, gin.H{"data": "false"})
				return
			}
			bind := utils.OauthBind{}
			result = db.Model(&utils.OauthBind{}).Where("addr = ?", address).First(&bind)
			// 判断返回结果里面address是不是空
			if bind != (utils.OauthBind{}) {
				result = db.Model(&bind).Where("addr = ?", address).Updates(map[string]interface{}{"gmail": profile.EmailAddress})
				if result.Error != nil {
					logger.Error("failed to update address:", zap.Error(result.Error))
					c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Update Error"))
					return
				}
			} else {
				result = db.Model(&utils.OauthBind{}).Create(&utils.OauthBind{Addr: address, Gmail: profile.EmailAddress})
				if result.Error != nil {
					logger.Error("failed to insert oauth_bind:", zap.Error(result.Error))
					c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Insert Error"))
					return
				}
			}
			c.JSON(http.StatusOK, gin.H{"data": "success"})
		}
	})

	r.POST("/oauth/login", func(c *gin.Context) {
		var requestBody utils.RequestLoginBody
		// 将请求体中的 JSON 数据绑定到结构体
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			// 处理绑定错误
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		code := requestBody.Code
		platformType := requestBody.PlatformType
		if code == "" || platformType == "" {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("参数错误"))
			return
		}
		if platformType == "github" {
			module.GithubLogin(c, code)
		} else {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("平台不支持"))
			return
		}
	})

	// github oauth
	r.GET("/oauth/github", func(c *gin.Context) {
		code := c.Query("code")
		source := c.Query("source")
		if code == "" {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("No authorization code provided."))
			return
		}
		logger.Info("github oauth认证", zap.String("code", code))
		logger.Info("github oauth source", zap.String("source", source))
		if source != "" { // 使用OAuth配置对象中定义的Exchange方法，通过code获取access token
			// 拼接https://transformer.knn3.xyz/ + source + /type=github&code= + code
			url := fmt.Sprintf("https://transformer.knn3.xyz/%s?type=github&code=%s", source, code)
			c.Redirect(http.StatusTemporaryRedirect, url)
		} else {
			c.Redirect(http.StatusTemporaryRedirect, "https://topscore.social/pass?type=github&code="+code)
		}
	})

	// github oauth
	r.GET("/oauth/discord", func(c *gin.Context) {
		code := c.Query("code")
		if code == "" {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("No authorization code provided."))
			return
		}
		logger.Info("discord oauth认证", zap.String("code", code))

		c.Redirect(http.StatusTemporaryRedirect, "https://topscore.social/pass?type=discord&code="+code)
	})

	r.GET("/oauth/gmail", func(c *gin.Context) {
		code := c.Query("code")
		state := c.Query("state")

		// logger.Info("gmail state", zap.String("state", "state"))
		// logger.Info("gmail url", zap.String("state", Request.URL.String()))

		if code == "" || state == "" {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("No authorization code provided."))
			return
		}
		logger.Info("gmail oauth认证", zap.String("code", code))

		// knexus gmail login
		decodedURL, err := url.QueryUnescape(state)

		if err != nil {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("authorization error state"))
			return
		}

		stateArr := strings.Split(decodedURL, "$")

		logger.Info("gmail state arr", zap.Any("stateArr", stateArr))
		if stateArr[0] == "knexus" || stateArr[0] == "knexus_early" {
			success := stateArr[1]
			fail := stateArr[2]

			profile, err := module.GetGmailProfileByKnexus(code)
			if err != nil {
				c.AbortWithError(http.StatusBadRequest, fmt.Errorf("authorization error code"))
				return
			}

			source := ""
			if stateArr[0] == "knexus" {
				source = "normal"
			}
			if stateArr[0] == "knexus_early" {
				source = "early"
			}

			accessToken, err := module.GetAccessToken(profile.EmailAddress, source)
			if err != nil {
				c.Redirect(http.StatusMovedPermanently, strings.Replace(fail, "fail=", "", 1))
				return
			}

			c.Redirect(http.StatusMovedPermanently, strings.Replace(success, "success=", "", 1)+"?j="+base64.StdEncoding.EncodeToString([]byte(accessToken)))
			return
		}

		c.Redirect(http.StatusTemporaryRedirect, "https://topscore.social/pass?type=gmail&code="+code)
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
