package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

type HitScore struct {
	EntryID int64
	Score   float32
}

type searchResult struct {
	Res []HitScore
	Err error
}

func (s *Storage) Search(ctx context.Context, tags []string, queryEmbed []float32) (Entry, error) {
	tagCh := make(chan searchResult, 1)
	visCh := make(chan searchResult, 1)
	textCh := make(chan searchResult, 1)

	var (
		tagRes, visRes, textRes searchResult
		gotTag, gotVis, gotText bool
	)

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if s.tagSearchWeight != 0 {
		go func() { hits, err := s.TagSearch(subCtx, tags); tagCh <- searchResult{hits, err} }()
	} else {
		tagCh <- searchResult{}
	}

	if s.visualSearchWeight != 0 {
		go func() { hits, err := s.VisualSearch(subCtx, queryEmbed); visCh <- searchResult{hits, err} }()
	} else {
		visCh <- searchResult{}
	}

	if s.textSearchWeight != 0 {
		go func() { hits, err := s.TextSearch(subCtx, queryEmbed); textCh <- searchResult{hits, err} }()
	} else {
		textCh <- searchResult{}
	}

	for !gotTag || !gotVis || !gotText {
		select {
		case tagRes = <-tagCh:
			gotTag = true
			if tagRes.Err != nil {
				cancel()
				return Entry{}, tagRes.Err
			}
		case visRes = <-visCh:
			gotVis = true
			if visRes.Err != nil {
				cancel()
				return Entry{}, visRes.Err
			}
		case textRes = <-textCh:
			gotText = true
			if textRes.Err != nil {
				cancel()
				return Entry{}, textRes.Err
			}
		case <-ctx.Done():
			return Entry{}, ctx.Err()
		}
	}

	entries := make(map[int64]float32)

	for _, v := range tagRes.Res {
		entries[v.EntryID] += v.Score * s.tagSearchWeight
	}
	for _, v := range visRes.Res {
		entries[v.EntryID] += v.Score * s.visualSearchWeight
	}
	for _, v := range textRes.Res {
		entries[v.EntryID] += v.Score * s.textSearchWeight
	}

	max := float32(math.Inf(-1))
	var resID int64

	for id, value := range entries {
		if value > max {
			max = value
			resID = id
		}
	}

	res, err := s.GetEntry(resID)
	if err != nil {
		return Entry{}, err
	}

	fmt.Printf("Score hit: %v. ", max)
	return res, nil
}

func (s *Storage) TagSearch(ctx context.Context, tags []string) ([]HitScore, error) {
	result := make([]HitScore, 0)

	if len(tags) <= 0 {
		return result, nil
	}

	placeholder := fmt.Sprintf("(%s)", strings.Repeat("?,", len(tags)-1)+"?")

	tagsArgs := make([]any, len(tags))
	for i, v := range tags {
		tagsArgs[i] = v
	}

	query := fmt.Sprintf(`
		SELECT entry_id, COUNT(*) AS matches
		FROM entry_tags
		WHERE tag_id IN (
			SELECT id
			FROM tags
			WHERE name IN %s
		)
		GROUP BY entry_id
		ORDER BY matches DESC;
	`, placeholder)

	rows, err := s.db.QueryContext(ctx, query, tagsArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entryID int64
		var matches int

		err = rows.Scan(&entryID, &matches)
		if err != nil {
			return nil, err
		}

		result = append(result, HitScore{
			EntryID: entryID,
			Score:   float32(matches) / float32(len(tags)),
		})
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return result, nil
}

func (s *Storage) VisualSearch(ctx context.Context, embedding []float32) ([]HitScore, error) {
	return s.embeddingSearch(ctx, embedding, "visual_embeddings")
}

func (s *Storage) TextSearch(ctx context.Context, embedding []float32) ([]HitScore, error) {
	return s.embeddingSearch(ctx, embedding, "ocr_embeddings")
}

func (s *Storage) embeddingSearch(ctx context.Context, embedding []float32, tableName string) ([]HitScore, error) {
	result := make([]HitScore, 0)

	query := fmt.Sprintf(`
			SELECT rowid, distance
			FROM %s
			WHERE embedding MATCH ?
			ORDER BY distance
			LIMIT 50;`, tableName)

	embedJSON, err := json.Marshal(embedding)
	if err != nil {
		return result, err
	}

	rows, err := s.db.QueryContext(ctx, query, string(embedJSON))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entryID int64
		var distance float32

		err = rows.Scan(&entryID, &distance)
		if err != nil {
			return nil, err
		}

		result = append(result, HitScore{
			EntryID: entryID,
			Score:   1 - distance,
		})
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return result, nil
}
