package module

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

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

// init
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
	url := stackoverflowConfig.AuthCodeURL("state")
	logger.Info("Stackoverflow oauth AuthCodeURL", zap.String("url", url))

	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}

// CallBack //
//
//	@receiver sf
//	@param c
func (sf Stackoverflow) CallBack(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("No authorization code provided."))
		return
	}
	logger.Info("stackoverflow oauth", zap.String("code", code))

	c.Redirect(http.StatusTemporaryRedirect, "https://topscore.social/pass/succss?type=stackexchange&code="+code)
}

// Bind
//
//	@receiver sf
//	@param c
//	@param code
//	@param address
func (sf Stackoverflow) Bind(c *gin.Context, code string, address string) {

	logger.Info("Stackoverflow CallBack", zap.String("code", code))

	// get token
	token, err := stackoverflowConfig.Exchange(c, code)

	logger.Info("Stackoverflow token", zap.Any("token", token))
	if err != nil {
		logger.Error("Stackoverflow failed to exchange token:", zap.Error(err))
		// Redirect error page
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("error code"))
		return
	}

	client := stackoverflowConfig.Client(c, token)
	userInfo, err := sf.UserInfo(client, token.AccessToken)
	if err != nil {
		logger.Error("failed to get user info:", zap.Error(err))
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("error userInfo"))
		return
	}
	// logger.Info("Stackoverflow userInfo", zap.Any("userInfo", &userInfo))

	logger.Info("Stackoverflow userInfo", zap.Any("userInfo", userInfo))

	// get stackexchange id
	stackexchangeId := userInfo["items"].([]interface{})[0].(map[string]interface{})["account_id"].(float64)

	logger.Info("Stackoverflow stackexchangeId", zap.Float64("stackexchangeId", stackexchangeId))

	db := utils.GetDB()

	addr := utils.OauthBind{}
	result := db.Model(&utils.OauthBind{}).Where("exchange = ?", stackexchangeId).First(&addr)
	if addr != (utils.OauthBind{}) {
		logger.Error("stackoverflow has bound:", zap.Error(result.Error))
		c.JSON(http.StatusOK, gin.H{"data": "stackoverflow has bound"})
		return
	}

	bind := utils.OauthBind{}
	result = db.Model(&utils.OauthBind{}).Where("addr = ?", address).First(&bind)
	// check address
	if bind != (utils.OauthBind{}) {
		result = db.Model(&bind).Where("addr = ?", address).Updates(map[string]interface{}{"exchange": stackexchangeId})
		if result.Error != nil {
			logger.Error("failed to update address:", zap.Error(result.Error))
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("stackexchange update error"))
			return
		}
	} else {
		result = db.Model(&utils.OauthBind{}).Create(&utils.OauthBind{Addr: address, Exchange: strconv.FormatFloat(stackexchangeId, 'f', -1, 64)})
		if result.Error != nil {
			logger.Error("failed to insert oauth_bind:", zap.Error(result.Error))
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("insert error"))
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": "success"})

}

/*
	{
		"userInfo":{
		   "has_more":false,
		   "items":[
			  {
				 "account_id":xxx,
				 "badge_counts":{
					"bronze":0,
					"gold":0,
					"silver":0
				 },
				 "creation_date":xxx,
				 "display_name":"xxxx",
				 "is_employee":false,
				 "last_access_date":1684201040,
				 "link":"https://stackoverflow.com/users/10267509/xxx",
				 "profile_image":"xxx",
				 "reputation":1,
				 "reputation_change_day":0,
				 "reputation_change_month":0,
				 "reputation_change_quarter":0,
				 "reputation_change_week":0,
				 "reputation_change_year":0,
				 "user_id":xxx,
				 "user_type":"registered"
			  }
		   ],
		   "quota_max":10000,
		   "quota_remaining":9999
		}
	 }
*/

// UserInfo
//
//	@receiver sf
//	@param client
//	@param accessToken
//	@return map[string]interface{}
//	@return error
func (sf Stackoverflow) UserInfo(client *http.Client, accessToken string) (map[string]interface{}, error) {

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

// decodeResponse
//
//	@param resp
//	@param v
//	@return error
func decodeResponse(resp *http.Response, v interface{}) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to decode response body: %w", err)
	}

	return nil
}
