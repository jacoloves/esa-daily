package main

import (
	"fmt"
	"log"
	"os"
	"time"

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

func updatePost(team, token string, number int, name, exisitingBody, newEntry string) error {
	client := resty.New()

	updatedBody := exisitingBody + "\n" + newEntry

	reqBody := map[string]interface{}{
		"post": map[string]interface{}{
			"name":    name,
			"body_md": updatedBody,
			"wip":     true,
		},
	}

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+token).
		SetHeader("Congent-Type", "application/json").
		SetBody(reqBody).
		Put(fmt.Sprintf("https://api.esa.io/v1/teams/%s/posts/%d", team, number))
	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("Article update failed", resp.Status())
	}

	fmt.Println("Added to the article. ✅")
	return nil
}

func creaatePostFromTemplate(team, token, category, name, templateFullName string) error {
	client := resty.New()

	reqBody := map[string]interface{}{
		"post": map[string]interface{}{
			"name":                    name,
			"category":                category,
			"wip":                     true,
			"template_post_full_name": templateFullName,
		},
	}

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+token).
		SetHeader("Content-Type", "application/json").
		SetBody(reqBody).
		Post(fmt.Sprintf("https://api.esa.io/v1/teams/%s/posts", team))
	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("Failed to create article: %s", resp.Status())
	}

	fmt.Println("Created an article from a template. ✅")
	return nil
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

	now := time.Now()
	year := now.Format("06")
	month := now.Format("01")
	day := now.Format("02")

	category := fmt.Sprintf("dairy/%s/%s/%s", year, month, day)
	name := "dairy"
	fullName := fmt.Sprintf("%s/%s", category, name)
	template := fmt.Sprintf("Templates/%s/%s", category, name)

	timestamp := now.Format("15:04")
	message := "I will wirte some Go code"
	newEntry := fmt.Sprintf("%s %s", timestamp, message)

	postResp, err := getPostByFullName(team, token, fullName)
	if err != nil {
		log.Fatalf("Failed to retrieve article: %v", err)
	}

	if len(postResp.Posts) > 0 {
		post := postResp.Posts[0]
		err = updatePost(team, token, post.Number, post.Name, post.BodyMd, newEntry)
		if err != nil {
			log.Fatalf("update error: %v", err)
		}
	} else {
		fmt.Println("No article exists. Create a new article from the template.")

		err := creaatePostFromTemplate(team, token, category, name, template)
		if err != nil {
			log.Fatalf("Error creating from template: %v", err)
		}

		postResp, err := getPostByFullName(team, token, fullName)
		if err != nil || len(postResp.Posts) == 0 {
			log.Fatalf("Filed to retrieve newly created post")
		}
		post := postResp.Posts[0]
		err = updatePost(team, token, post.Number, post.Name, post.BodyMd, newEntry)
		if err != nil {
			log.Fatalf("Update after create error: %v", err)
		}
	}
}
