package module

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmail "google.golang.org/api/gmail/v1"
)

var (
	oauthConfig *oauth2.Config
	oauthState  = "random-string" // 可以使用随机生成的字符串
)

var clientGamilID, clientGmailSecret, redirectGmailURI string

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GMAIL_ID"),                       // 替换为实际的客户端ID
		ClientSecret: os.Getenv("GMAIL_SECRET"),                   // 替换为实际的客户端密钥
		RedirectURL:  "https://knn3-gateway.knn3.xyz/oauth/gmail", // 替换为实际的回调URL
		Scopes: []string{
			gmail.GmailReadonlyScope,
		},
		Endpoint: google.Endpoint,
	}
}

func GetGmailProfile(code string) (*gmail.Profile, error) {
	token, err := oauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		fmt.Println("err:", err)
		return nil, err
	}
	client := oauthConfig.Client(oauth2.NoContext, token)
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
