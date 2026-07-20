package aimanip

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
)

type EmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type EmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func Embed(input []string) ([][]float32, error) {
	modelName := "bge-m3"

	reqBody := EmbedRequest{
		Model: modelName,
		Input: input,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("Error occurred while marshaling request body:%v", err)
	}

	url := "http://localhost:11434/api/embed"

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Error occurred while sending request to the model:%v", err)
	}
	defer resp.Body.Close()

	var embedResp EmbedResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error occurred while reading response body:%v", err)
	}

	err = json.Unmarshal(body, &embedResp)
	if err != nil {
		return nil, fmt.Errorf("Error occurred while unmarshaling response body:%v", err)
	}

	return embedResp.Embeddings, nil
}

// for testing
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64

	for i := range a {
		ax := float64(a[i])
		bx := float64(b[i])
		dot += ax * bx
		normA += ax * ax
		normB += bx * bx
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}
