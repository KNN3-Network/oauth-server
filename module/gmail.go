package module

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmail "google.golang.org/api/gmail/v1"
)

var (
	oauthConfig *oauth2.Config
	oauthState  = "random-string" // 可以使用随机生成的字符串

	oauthKnexusConfig *oauth2.Config
)

var clientGamilID, clientGmailSecret, redirectGmailURI string

func init() {
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatal("Error loading .env file")
	// }
	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GMAIL_ID"),                       // 替换为实际的客户端ID
		ClientSecret: os.Getenv("GMAIL_SECRET"),                   // 替换为实际的客户端密钥
		RedirectURL:  "https://knn3-gateway.knn3.xyz/oauth/gmail", // 替换为实际的回调URL
		Scopes: []string{
			gmail.GmailReadonlyScope,
		},
		Endpoint: google.Endpoint,
	}

	oauthKnexusConfig = &oauth2.Config{
		ClientID:     os.Getenv("KNEXUS_GMAIL_ID"),           // 替换为实际的客户端ID
		ClientSecret: os.Getenv("KNEXUS_GMAIL_SECRET"),       // 替换为实际的客户端密钥
		RedirectURL:  os.Getenv("KNEXUS_GMAIL_REDIRECT_URL"), // 替换为实际的回调URL
		Scopes: []string{
			gmail.GmailReadonlyScope,
		},
		Endpoint: google.Endpoint,
	}

	url := oauthKnexusConfig.AuthCodeURL("knexus$success=https://knexus.xyz$fail=https://knexus.xyz")
	logger.Info("gmail oauth AuthCodeURL", zap.String("url", url))

}

func GetGmailProfile(code string) (*gmail.Profile, error) {
	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		fmt.Println("err:", err)
		return nil, err
	}
	client := oauthConfig.Client(context.Background(), token)
	gmailService, err := gmail.New(client)
	if err != nil {
		fmt.Println("err:", err)
		return nil, err
	}

	profile, err := gmailService.Users.GetProfile("me").Do()
	if err != nil {
		fmt.Println("err:", err)
		return nil, err
	}
	return profile, nil

}

func GetGmailProfileByKnexus(code string) (*gmail.Profile, error) {
	token, err := oauthKnexusConfig.Exchange(context.Background(), code)
	if err != nil {
		fmt.Println("err:", err)
		return nil, err
	}
	client := oauthKnexusConfig.Client(context.Background(), token)
	gmailService, err := gmail.New(client)
	if err != nil {
		fmt.Println("err:", err)
		return nil, err
	}

	profile, err := gmailService.Users.GetProfile("me").Do()
	if err != nil {
		fmt.Println("err:", err)
		return nil, err
	}
	return profile, nil

}

// GetAccessToken GetAccessToken
//
//	@param email
//	@param source
//	@return string
//	@return error
func GetAccessToken(email string, source string) (string, error) {
	url := os.Getenv("KNEXUS_API")
	method := "POST"

	payload := strings.NewReader(`{"gmail": "` + email + `","type": "` + source + `"}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println("err:", err)
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("err:", err)
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("err:", err)
		return "", err
	}
	fmt.Println(string(body))

	return string(body), nil

}
