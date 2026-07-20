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

const (
	tagModelName = "qwen2.5vl"
	tagUrl       = "http://localhost:11434/api/chat"
)

type Message struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"`
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

type TagImageResponse struct {
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Text        string   `json:"text"`
}

type TagReqResponse struct {
	Tags []string `json:"tags"`
}

// TagImage takes an image path as input and returns a slice of tags, an image description, and an error if any occurs during the tagging process.
func TagImage(imagePath string) (TagImageResponse, error) {
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return TagImageResponse{}, fmt.Errorf("Error occured while opening image file:%v", err)
	}
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	reqBody := ChatRequest{
		Model: tagModelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: PromptTagPhoto,
				Images:  []string{base64Image},
			},
		},
		Stream: false,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return TagImageResponse{}, fmt.Errorf("Error occurred while marshaling request body:%v", err)
	}

	resp, err := http.Post(tagUrl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return TagImageResponse{}, fmt.Errorf("Error occurred while sending request to the model:%v", err)
	}
	defer resp.Body.Close()

	var chatResp ChatResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TagImageResponse{}, fmt.Errorf("Error occurred while reading response body:%v", err)
	}

	err = json.Unmarshal(body, &chatResp)
	if err != nil {
		return TagImageResponse{}, fmt.Errorf("Error occurred while unmarshaling response body:%v", err)
	}

	var tagResp TagImageResponse
	err = json.Unmarshal([]byte(chatResp.Message.Content), &tagResp)
	if err != nil {
		return TagImageResponse{}, fmt.Errorf("Error occurred while unmarshaling tag response:%v", err)
	}

	return tagResp, nil
}

func TagRequest(request string) ([]string, error) {
	reqBody := ChatRequest{
		Model: tagModelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: fmt.Sprintf(PromptTagRequest, request),
			},
		},
		Stream: false,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("Error occurred while marshaling request body:%v", err)
	}

	resp, err := http.Post(tagUrl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Error occurred while sending request to the model:%v", err)
	}
	defer resp.Body.Close()

	var chatResp ChatResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error occurred while reading response body:%v", err)
	}

	err = json.Unmarshal(body, &chatResp)
	if err != nil {
		return nil, fmt.Errorf("Error occurred while unmarshaling response body:%v", err)
	}

	var tagResp TagReqResponse

	err = json.Unmarshal([]byte(chatResp.Message.Content), &tagResp)
	if err != nil {
		return nil, fmt.Errorf("Error occurred while unmarshaling response body:%v", err)
	}

	return tagResp.Tags, nil
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
