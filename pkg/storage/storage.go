package storage

import (
	"fmt"

	bleve "github.com/blevesearch/bleve/v2"
)

type Storage struct {
	Index bleve.Index
	Path  string
}

type Entry struct {
	ID         string      `json:"id"`
	Path       string      `json:"path"`
	Tags       []string    `json:"tags"`
	Embeddings [][]float64 `json:"embeddings"`
}

func NewStorage(path string) (*Storage, error) {
	index, err := bleve.Open(path)
	if err == bleve.ErrorIndexPathDoesNotExist {
		mapping := bleve.NewIndexMapping()

		tagsField := bleve.NewTextFieldMapping()
		tagsField.Analyzer = "keyword"

		vectorField := bleve.NewVectorFieldMapping()
		vectorField.Dims = 1024
		//vectorField.Similarity =

		index, err = bleve.New(path, mapping)
	}
	if err != nil {
		return nil, fmt.Errorf("Error occurred while opening/creating index:%v", err)
	}
	return &Storage{Index: index, Path: path}, nil
}
