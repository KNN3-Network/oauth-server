package module

import (
	"log"
	"testing"

	"github.com/joho/godotenv"
)

func setup() {
	err := godotenv.Load("~/Desktop/project/treasury/oauth-server/.env")

	log.Print("setup")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func TestGetAccessToken(t *testing.T) {
	GetAccessToken()
}
