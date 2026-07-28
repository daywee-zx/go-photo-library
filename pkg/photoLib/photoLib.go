package photolib

import (
	"context"
	"fmt"

	"github.com/daywee-zx/go-photo-library/pkg/aimanip"
	"github.com/daywee-zx/go-photo-library/pkg/storage"
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
	DeleteEntry(int64) error

	GetEntry(context.Context, int64) (storage.Entry, error)
	GetEntryTags(context.Context, int64) ([]string, error)
	GetAvailableTags(context.Context) ([]string, error)

	Search(ctx context.Context, tags []string, queryEmbed []float32, topK int) ([]storage.Entry, error)
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

func (p *PhotoLib) AddImage(e storage.Entry) (int64, error) {
	tagResp, err := p.tagger.TagImage(e.Path)
	if err != nil {
		return 0, fmt.Errorf("Error tagging the image %s:%v", e.Path, err)
	}

	embeds, err := p.embedder.Embed([]string{tagResp.Description, tagResp.Text})
	if err != nil {
		return 0, fmt.Errorf("Error embedding image descriptions %s:%v", e.Path, err)
	}

	entry := storage.IndexedEntry{
		Entry:       e,
		VisualEmbed: embeds[0],
		TextEmbed:   embeds[1],
	}

	entry.Tags = append(e.Tags, tagResp.Tags...)
	entry.Description = tagResp.Description

	id, err := p.store.InsertEntry(entry)
	if err != nil {
		return 0, fmt.Errorf("Error during adding entry %s:%v", e.Path, err)
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

func (p *PhotoLib) Search(request string, topK int) ([]storage.Entry, error) {
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
			return nil, err
		}
	}

	p.UpdateSearchWeights()

	entries, err := p.store.Search(p.ctx, tags, embedding, topK)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (p *PhotoLib) InsertEntry(e storage.IndexedEntry) (int64, error) {
	return p.store.InsertEntry(e)
}

func (p *PhotoLib) DeleteEntry(id int64) error {
	return p.store.DeleteEntry(id)
}

func (p *PhotoLib) GetEntry(id int64) (storage.Entry, error) {
	return p.store.GetEntry(p.ctx, id)
}
