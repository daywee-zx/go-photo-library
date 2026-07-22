package photolib

import (
	"context"
	"fmt"
	"photoLibrary/pkg/aimanip"
	"photoLibrary/pkg/storage"
)

type Embedder interface {
	Embed(input []string) ([][]float32, error)
}

type Tagger interface {
	TagImage(path string) (aimanip.TagImageData, error)
	TagRequest(request string) ([]string, error)
}

type StorageBack interface {
	InsertEntry(storage.IndexedEntry) (int64, error)
	Search(ctx context.Context, tags []string, queryEmbed []float32) (storage.Entry, error)
	SetSearchWeights(tag, visual, text float32)
	Close() error
}

type SearchConfig struct {
	tagWeight    float32
	visualWeight float32
	textWeight   float32
}

type PhotoLib struct {
	store    StorageBack
	embedder Embedder
	tagger   Tagger
	config   SearchConfig

	ctx context.Context
}

func NewPhotoLib(ctx context.Context, store StorageBack, embedder Embedder, tagger Tagger, config SearchConfig) *PhotoLib {
	return &PhotoLib{
		store:    store,
		embedder: embedder,
		tagger:   tagger,
		config:   config,
		ctx:      ctx,
	}
}

func (p *PhotoLib) AddImage(path string) (int64, error) {
	tagResp, err := p.tagger.TagImage(path)
	if err != nil {
		return 0, fmt.Errorf("Error tagging the image %s:%v", path, err)
	}

	embeds, err := p.embedder.Embed([]string{tagResp.Description, tagResp.Text})
	if err != nil {
		return 0, fmt.Errorf("Error embedding image descriptions %s:%v", path, err)
	}

	entry := storage.IndexedEntry{
		Entry: storage.Entry{
			ID:   0,
			Path: path,
			Tags: tagResp.Tags,
		},
		VisualEmbed: embeds[0],
		TextEmbed:   embeds[1],
	}

	id, err := p.store.InsertEntry(entry)
	if err != nil {
		return 0, fmt.Errorf("Error during adding entry %s:%v", path, err)
	}

	return id, nil
}

func (p *PhotoLib) SetSearchWeights(s SearchConfig) {
	p.store.SetSearchWeights(s.tagWeight, s.visualWeight, s.textWeight)
}
func (p *PhotoLib) UpdateSearchWeights() {
	p.SetSearchWeights(p.config)
}
func NewSearchConfig(tag, visual, text float32) SearchConfig {
	return SearchConfig{tag, visual, text}
}

func (p *PhotoLib) Search(request string) (storage.Entry, error) {
	embeddingCh := make(chan []float32, 1)
	tagsCh := make(chan []string, 1)
	errorCh := make(chan error, 2)

	go func() {
		embeds := make([][]float32, 1)
		embeds, err := p.embedder.Embed([]string{request})
		embeddingCh <- embeds[0]
		errorCh <- err
	}()
	go func() {
		tags, err := p.tagger.TagRequest(request)
		tagsCh <- tags
		errorCh <- err
	}()

	embedding := <-embeddingCh
	tags := <-tagsCh
	for i := 0; i < 2; i++ {
		if err := <-errorCh; err != nil {
			return storage.Entry{}, err
		}
	}

	p.UpdateSearchWeights()

	entry, err := p.store.Search(p.ctx, tags, embedding)
	if err != nil {
		return storage.Entry{}, err
	}

	return entry, nil
}
