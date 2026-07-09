package aimanip

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Message struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type ChatResponse struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
}

type TagResponse struct {
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Text        string   `json:"text"`
}

// TagImage takes an image path as input and returns a slice of tags, an image description, and an error if any occurs during the tagging process.
func TagImage(imagePath string) (string, error) {
	modelName := "qwen2.5vl"

	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("Error occured while opening image file:%v", err)
	}
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	prompt, err := os.ReadFile("pkg/aimanip/tagger_prompt.txt")
	if err != nil {
		return "", fmt.Errorf("Error occurred while reading prompt file:%v", err)
	}
	promptStr := string(prompt)

	reqBody := ChatRequest{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: promptStr,
				Images:  []string{base64Image},
			},
		},
		Stream: false,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("Error occurred while marshaling request body:%v", err)
	}

	url := "http://localhost:11434/api/chat"

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("Error occurred while sending request to the model:%v", err)
	}
	defer resp.Body.Close()

	var chatResp ChatResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Error occurred while reading response body:%v", err)
	}

	err = json.Unmarshal(body, &chatResp)
	if err != nil {
		return "", fmt.Errorf("Error occurred while unmarshaling response body:%v", err)
	}

	return chatResp.Message.Content, nil
}

func parseResponse(response string) ([]string, string) {
	parts := strings.Split(response, ".")
	desc := strings.TrimSpace(parts[0])
	tags := strings.Split(parts[1], ",")

	for i, tag := range tags {
		tags[i] = strings.TrimSpace(tag)
	}

	return tags, desc
}
