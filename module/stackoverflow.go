package module

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/KNN3-Network/oauth-server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/stackoverflow"
)

var logger = utils.Logger

var (
	stackoverflowConfig *oauth2.Config
)

func init() {

	stackoverflowConfig = &oauth2.Config{
		ClientID:     os.Getenv("STACKOVERFLOW_CLIENT_ID"),
		ClientSecret: os.Getenv("STACKOVERFLOW_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("STACKOVERFLOW_REDIRECT_URL"),
		Scopes:       []string{}, // https://api.stackexchange.com/docs/authentication#scope
		Endpoint:     stackoverflow.Endpoint,
	}

	logger.Info("Stackoverflow oauth config", zap.Any("stackoverflowConfig", &stackoverflowConfig))

}

type Stackoverflow struct{}

// AuthCodeURL
//
//	@receiver sf
//	@param c
func (sf Stackoverflow) AuthCodeURL(c *gin.Context) {

	logger.Error("Stackoverflow failed to exchange token:")

	url := stackoverflowConfig.AuthCodeURL("state")
	logger.Info("Stackoverflow oauth AuthCodeURL", zap.String("url", url))

	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})

}

// CallBack
//
//	@receiver sc
//	@param c
func (sf Stackoverflow) CallBack(c *gin.Context) {
	code := c.Query("code")

	logger.Info("Stackoverflow CallBack", zap.String("code", code))
	// get token
	token, err := stackoverflowConfig.Exchange(c, code)

	logger.Info("Stackoverflow token", zap.Any("token", token))
	if err != nil {
		logger.Error("Stackoverflow failed to exchange token:", zap.Error(err))
		// Redirect error page
		c.Redirect(http.StatusTemporaryRedirect, "/error")
		return
	}

	client := stackoverflowConfig.Client(c, token)
	userInfo, err := sf.UserInfo(client, token.AccessToken)
	if err != nil {
		logger.Error("failed to get user info:", zap.Error(err))
		c.Redirect(http.StatusTemporaryRedirect, "/error")
		return
	}

	logger.Info("Stackoverflow userInfo", zap.Any("userInfo", &userInfo))

	// c.Redirect(http.StatusTemporaryRedirect, os.Getenv("SUCCESS_WEB_SITE"))
	c.JSON(http.StatusOK, gin.H{
		"message": "pong11",
	})
}

// UserInfo UserInfo
//
//	@receiver sf
//	@param client
//	@return map[string]interface{}
//	@return error
func (sf Stackoverflow) UserInfo(client *http.Client, accessToken string) (map[string]interface{}, error) {

	/*
		resp, err := client.Get("https://api.stackexchange.com/2.3/me?key=" + os.Getenv("STACKOVERFLOW_APPS_KEY") + "&site=stackoverflow")

		if err != nil {
			fmt.Println(err)
			return nil, fmt.Errorf("get user info request failed: %w", err)
		}

		defer resp.Body.Close()

		logger.Info("Stackoverflow client", zap.Any("Body", client.Transport))
		logger.Info("Stackoverflow Body", zap.Any("Body", resp.Body))
		var userInfo map[string]interface{}
		if err := decodeResponse(resp, &userInfo); err != nil {
			return nil, err
		}

		return userInfo, nil


	*/

	req, err := http.NewRequest("GET", "https://api.stackexchange.com/2.3/me", nil)

	q := req.URL.Query()
	q.Add("key", os.Getenv("STACKOVERFLOW_APPS_KEY"))
	q.Add("site", "stackoverflow")
	q.Add("access_token", accessToken)

	req.URL.RawQuery = q.Encode()

	logger.Info("Stackoverflow oauth req.URL.String", zap.String("url", req.URL.String()))

	if err != nil {
		return nil, fmt.Errorf("get user info request failed: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get user info request failed: %w", err)
	}

	defer resp.Body.Close()

	logger.Info("Stackoverflow Body", zap.Any("Body", resp.Body))
	var userInfo map[string]interface{}
	if err := decodeResponse(resp, &userInfo); err != nil {
		return nil, err
	}

	return userInfo, nil

}

func decodeResponse(resp *http.Response, v interface{}) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to decode response body: %w", err)
	}

	return nil
}
