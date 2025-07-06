package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
)

type EsaResponse struct {
	Posts []struct {
		Number int    `json:"number"`
		Name   string `json:"name"`
		BodyMd string `json:"body_md"`
		Wip    bool   `json:"wip"`
	} `json:"posts"`
}

func getPostByFullName(team, token, fullName string) (*EsaResponse, error) {
	client := resty.New()

	resp, err := client.R().
		SetQueryParams(map[string]string{
			"q": fmt.Sprintf("full_name:%s", fullName),
		}).
		SetHeader("Authorization", "Bearer "+token).
		SetHeader("Accept", "application/json").
		SetResult(&EsaResponse{}).
		Get(fmt.Sprintf("https://api.esa.io/v1/teams/%s/posts", team))
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("API error: %s", resp.Status())
	}

	return resp.Result().(*EsaResponse), nil
}

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

	fullName := "dairy/25/07/06/dairy"

	postResp, err := getPostByFullName(team, token, fullName)
	if err != nil {
		log.Fatalf("Failed to retrieve article: %v", err)
	}

	if len(postResp.Posts) == 0 {
		fmt.Println("No articles found")
	} else {
		post := postResp.Posts[0]
		fmt.Println("Article:", post.Number)
		fmt.Println("WIP:", post.Wip)
		fmt.Println("Detail:")
		fmt.Println(post.BodyMd)
	}
}
