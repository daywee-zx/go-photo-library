package aimanip

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Embedder struct {
	ModelName string
	URL       string
}

type EmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type EmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func (e Embedder) Embed(input []string) ([][]float32, error) {
	modelName := e.ModelName

	reqBody := EmbedRequest{
		Model: modelName,
		Input: input,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("Error occurred while marshaling request body:%v", err)
	}

	url := e.URL

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
