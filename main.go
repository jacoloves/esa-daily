package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func updatePost(team, token string, number int, name, existingBody, newEntry string) error {
	client := resty.New()

	updatedBody := existingBody + "\n" + newEntry

	reqBody := map[string]interface{}{
		"post": map[string]interface{}{
			"name":    name,
			"body_md": updatedBody,
			"wip":     true,
		},
	}

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+token).
		SetHeader("Content-Type", "application/json").
		SetBody(reqBody).
		Put(fmt.Sprintf("https://api.esa.io/v1/teams/%s/posts/%d", team, number))
	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("Article update failed", resp.Status())
	}

	return nil
}

func createPostFromTemplate(team, token, category, name, templateFullName string) error {
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

	return nil
}

// Bubble Tea model and message
type model struct {
	textInput textinput.Model
	team      string
	token     string
	messages  []string
	quitting  bool
}

type postResultMsg struct {
	success bool
	err     error
	message string
}

// style define
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 2).
			Width(80)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))
)

func initialModel(team, token string) model {
	ti := textinput.New()
	ti.Placeholder = "メッセージを入力してください..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 70

	return model{
		textInput: ti,
		team:      team,
		token:     token,
		messages:  []string{},
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			value := strings.TrimSpace(m.textInput.Value())

			if strings.ToLower(value) == "exit" || strings.ToLower(value) == "quit" || strings.ToLower(value) == "q" {
				m.quitting = true
				return m, tea.Quit
			}

			if value != "" {
				m.textInput.SetValue("")
				return m, m.postMessage(value)
			}
		}

	case postResultMsg:
		if msg.success {
			m.messages = append(m.messages, fmt.Sprintf("✅ 投稿完了: %s", msg.message))
		} else {
			m.messages = append(m.messages, fmt.Sprintf("❌ エラー: %v", msg.err))
		}

		if len(m.messages) > 5 {
			m.messages = m.messages[len(m.messages)-5:]
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) postMessage(message string) tea.Cmd {
	return func() tea.Msg {
		err := handlePost(m.team, m.token, message)
		return postResultMsg{
			success: err == nil,
			err:     err,
			message: message,
		}
	}
}

func (m model) View() string {
	if m.quitting {
		return promptStyle.Render("👋 またね！ \n")
	}

	title := titleStyle.Render("🔥 Esa Diary CLI")

	var history strings.Builder
	if len(m.messages) > 0 {
		history.WriteString("📝 最近の投稿:\n")

		start := 0
		if len(m.messages) > 5 {
			start = len(m.messages) - 5
		}

		for i := start; i < len(m.messages); i++ {
			msg := m.messages[i]
			if strings.Contains(msg, "✅") {
				history.WriteString(successStyle.Render(msg) + "\n")
			} else {
				history.WriteString(errorStyle.Render(msg) + "\n")
			}
		}
		history.WriteString("\n")
	}

	prompt := promptStyle.Render("📝 メッセージ: ") + m.textInput.View()
	help := helpStyle.Render("Enter: 投稿 | Ctrl+C/Esc: 終了 | exit/quit/q: 終了")

	content := fmt.Sprintf("%s\n\n%s%s\n%s", title, history.String(), prompt, help)

	return boxStyle.Render(content)
}

func handlePost(team, token, message string) error {
	now := time.Now()
	year := now.Format("06")
	month := now.Format("01")
	day := now.Format("02")

	category := fmt.Sprintf("dairy/%s/%s/%s", year, month, day)
	name := "dairy"
	fullName := fmt.Sprintf("%s/%s", category, name)
	template := fmt.Sprintf("Templates/%s/%s", category, name)

	timestamp := now.Format("15:04")
	newEntry := fmt.Sprintf("%s %s", timestamp, message)

	postResp, err := getPostByFullName(team, token, fullName)
	if err != nil {
		return fmt.Errorf("failed to retrieve article: %v", err)
	}

	if len(postResp.Posts) > 0 {
		post := postResp.Posts[0]
		err = updatePost(team, token, post.Number, post.Name, post.BodyMd, newEntry)
		if err != nil {
			return fmt.Errorf("update error: %v", err)
		}
	} else {
		err := createPostFromTemplate(team, token, category, name, template)
		if err != nil {
			return fmt.Errorf("error creating from template: %v", err)
		}

		time.Sleep(2 * time.Second)

		var post struct {
			Number int
			Name   string
			BodyMd string
		}

		for i := 0; i < 3; i++ {
			postResp, err := getPostByFullName(team, token, fullName)
			if err == nil && len(postResp.Posts) > 0 {
				post.Number = postResp.Posts[0].Number
				post.Name = postResp.Posts[0].Name
				post.BodyMd = postResp.Posts[0].BodyMd
				break
			}

			if i < 2 {
				time.Sleep(1 * time.Second)
			} else {
				return fmt.Errorf("failed to retrieve newly created post after %d attempts", i+1)
			}
		}

		err = updatePost(team, token, post.Number, post.Name, post.BodyMd, newEntry)
		if err != nil {
			return fmt.Errorf("update after create error: %v", err)
		}
	}

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

	p := tea.NewProgram(
		initialModel(team, token),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("エラーが発生しました: %v", err)
		os.Exit(1)
	}
}
