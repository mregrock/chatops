package yandexgpt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
	"github.com/sheeiavellie/go-yandexgpt"
)

func init() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: .env file not found or error loading: %v\n", err)
	}
}

func SendMessage(text string) (string, error) {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	systemPrompt, err := os.ReadFile(filepath.Join(dir, "system_prompt.txt"))
	if err != nil {
		return "", fmt.Errorf("failed to read system prompt: %w", err)
	}
	fmt.Println(string(systemPrompt))

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("API_KEY not found in environment")
	}

	catalogID := os.Getenv("API_CATALOG")
	if catalogID == "" {
		return "", fmt.Errorf("API_CATALOG not found in environment")
	}

	client := yandexgpt.NewYandexGPTClientWithAPIKey(apiKey)

	request := yandexgpt.YandexGPTRequest{
		ModelURI: yandexgpt.MakeModelURI(catalogID, yandexgpt.YandexGPTModelLite),
		CompletionOptions: yandexgpt.YandexGPTCompletionOptions{
			Stream:      false,
			Temperature: 0.5,
			MaxTokens:   200,
		},
		Messages: []yandexgpt.YandexGPTMessage{
			{
				Role: yandexgpt.YandexGPTMessageRoleSystem,
				Text: string(systemPrompt),
			},
			{
				Role: yandexgpt.YandexGPTMessageRoleUser,
				Text: text,
			},
		},
	}

	response, err := client.GetCompletion(context.Background(), request)
	if err != nil {
		return "", fmt.Errorf("failed to get GPT completion: %w", err)
	}

	if len(response.Result.Alternatives) == 0 {
		return "", fmt.Errorf("no response alternatives received")
	}

	return response.Result.Alternatives[0].Message.Text, nil
}
