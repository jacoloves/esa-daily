package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	token := os.Getenv("ESA_API_TOKEN")
	team := os.Getenv("ESA_TEAM_NAME")

	if token == "" || team == "" {
		log.Fatal("API token or team name has not been set.")
	}

	fmt.Println("Ready: Token and team name have been loaded.")
}
