package aimanip

import (
	"bytes"
	"context"
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

func (e Embedder) Embed(ctx context.Context, input []string) ([][]float32, error) {
	modelName := e.ModelName

	reqBody := EmbedRequest{
		Model: modelName,
		Input: input,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("Error occurred while marshaling request body:%v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.URL, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
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
