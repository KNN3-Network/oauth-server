package utils

// import (
// 	"encoding/json"
// 	"io/ioutil"
// 	"log"
// 	"net/http"
// 	"net/url"
// 	"os"

// 	discord "github.com/bwmarrin/discordgo"
// 	"github.com/joho/godotenv"
// 	"golang.org/x/oauth2"
// )

// var clientGamilID, clientGmailSecret, redirectGmailURI string

// func init() {
// 	err := godotenv.Load()
// 	if err != nil {
// 		log.Fatal("Error loading .env file")
// 	}
// 	clientGamilID = os.Getenv("GMAIL_ID")
// 	clientGmailSecret = os.Getenv("GMAIL_SECRET")
// 	redirectGmailURI = "http://localhost:8001/oauth/gmail"
// }

// func ExchangeGmailCodeForToken(code string) (*oauth2.Token, error) {
// 	form := url.Values{}
// 	form.Add("client_id", clientID)
// 	form.Add("client_secret", clientSecret)
// 	form.Add("grant_type", "authorization_code")
// 	form.Add("code", code)
// 	form.Add("redirect_uri", redirectURI)

// 	resp, err := http.PostForm("https://discordapp.com/api/oauth2/token", form)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var token oauth2.Token
// 	err = json.Unmarshal(body, &token)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &token, nil
// }

// func FetchUser(token *oauth2.Token) (*discord.User, error) {
// 	req, err := http.NewRequest("GET", "https://discordapp.com/api/users/@me", nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var user discord.User
// 	err = json.Unmarshal(body, &user)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &user, nil
// }
