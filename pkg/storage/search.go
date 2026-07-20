package storage

import (
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

func (s *Storage) Search(tags []string, queryEmbed []float32) (Entry, error) {
	tagCh := make(chan searchResult, 1)
	visCh := make(chan searchResult, 1)
	textCh := make(chan searchResult, 1)

	if s.tagSearchWeight != 0 {
		go func() { hits, err := s.TagSearch(tags); tagCh <- searchResult{hits, err} }()
	} else {
		tagCh <- searchResult{}
	}

	if s.visualSearchWeight != 0 {
		go func() { hits, err := s.VisualSearch(queryEmbed); visCh <- searchResult{hits, err} }()
	} else {
		visCh <- searchResult{}
	}

	if s.textSearchWeight != 0 {
		go func() { hits, err := s.TextSearch(queryEmbed); textCh <- searchResult{hits, err} }()
	} else {
		textCh <- searchResult{}
	}

	tagRes := <-tagCh
	visRes := <-visCh
	textRes := <-textCh

	if tagRes.Err != nil {
		return Entry{}, tagRes.Err
	}
	if visRes.Err != nil {
		return Entry{}, visRes.Err
	}
	if textRes.Err != nil {
		return Entry{}, textRes.Err
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

func (s *Storage) TagSearch(tags []string) ([]HitScore, error) {
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

	rows, err := s.db.Query(query, tagsArgs...)
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
		return nil, err
	}

	return result, nil
}

func (s *Storage) VisualSearch(embedding []float32) ([]HitScore, error) {
	return s.embeddingSearch(embedding, "visual_embeddings")
}

func (s *Storage) TextSearch(embedding []float32) ([]HitScore, error) {
	return s.embeddingSearch(embedding, "ocd_embeddings")
}

func (s *Storage) embeddingSearch(embedding []float32, tableName string) ([]HitScore, error) {
	result := make([]HitScore, 0)

	query := fmt.Sprintf(`
		SELECT rowid, distance
		FROM %s
		WHERE embedding MATCH ?
		ORDER BY distance
		LIMIT 50;`, tableName)

	embedJSON, err := json.Marshal(embedding)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(query, string(embedJSON))
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
		return nil, err
	}

	return result, nil
}
