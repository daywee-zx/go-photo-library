package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"
)

const (
	defaultTagWeight    float32 = 0.2
	defaultVisualWeight float32 = 0.4
	defaultTextWeight   float32 = 0.4
)

type Storage struct {
	db *sql.DB

	tagSearchWeight    float32
	visualSearchWeight float32
	textSearchWeight   float32
}

type Entry struct {
	ID   int64    `json:"id"`
	Path string   `json:"path"`
	Tags []string `json:"tags"`
}

type IndexedEntry struct {
	Entry

	VisualEmbed []float32 `json:"visual_embedding"`
	TextEmbed   []float32 `json:"ocr_embedding"`
}

func NewStorage(path string) (*Storage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	storage := &Storage{
		db:                 db,
		tagSearchWeight:    defaultTagWeight,
		visualSearchWeight: defaultVisualWeight,
		textSearchWeight:   defaultTextWeight,
	}
	return storage, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) Init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		)
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS entry_tags (
			entry_id INTEGER NOT NULL,
			tag_id INTEGER NOT NULL,
			PRIMARY KEY (entry_id, tag_id),
			FOREIGN KEY (entry_id) REFERENCES entries(id),
			FOREIGN KEY (tag_id) REFERENCES tags(id)
		)
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS visual_embeddings 
		USING vec0(
		embedding float[1024])
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS ocr_embeddings 
		USING vec0(
		embedding float[1024])
	`)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) InsertEntry(e IndexedEntry) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO entries (path) VALUES (?)
	`, e.Path)
	if err != nil {
		return 0, err
	}

	entryID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	err = insertTags(tx, entryID, e.Tags)
	if err != nil {
		return 0, err
	}

	visualEmbedJSON, err := json.Marshal(e.VisualEmbed)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		INSERT INTO visual_embeddings (rowid, embedding) VALUES (?, ?)
	`, entryID, string(visualEmbedJSON))
	if err != nil {
		return 0, err
	}

	textEmbedJSON, err := json.Marshal(e.TextEmbed)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		INSERT INTO ocr_embeddings (rowid, embedding) VALUES (?, ?)
	`, entryID, string(textEmbedJSON))
	if err != nil {
		return 0, err
	}

	return entryID, tx.Commit()
}

func insertTags(tx *sql.Tx, entryID int64, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	placeholders := make([]string, len(tags))
	for i := range tags {
		placeholders[i] = "(?)"
	}

	placeholder := strings.Join(placeholders, ",")

	args := make([]any, len(tags))
	for i, tag := range tags {
		args[i] = tag
	}

	query := fmt.Sprintf(`
		INSERT OR IGNORE INTO tags (name) VALUES %s
	`, placeholder)

	_, err := tx.Exec(query, args...)
	if err != nil {
		return err
	}

	placeholder = fmt.Sprintf("(%s)", strings.Repeat("?,", len(tags)-1)+"?")

	query = fmt.Sprintf(`
		INSERT INTO entry_tags (entry_id, tag_id)
		SELECT ?, id FROM tags WHERE name IN %s
	`, placeholder)

	entryArgs := make([]any, 1)
	entryArgs[0] = entryID
	entryArgs = append(entryArgs, args...)
	_, err = tx.Exec(query, entryArgs...)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) GetEntry(entryID int64) (Entry, error) {
	query := `
		SELECT path 
		FROM entries
		WHERE id = ?
	`

	row := s.db.QueryRow(query, entryID)

	var path string

	err := row.Scan(&path)
	if err != nil {
		return Entry{}, err
	}

	//TODO return tags
	return Entry{
		entryID,
		path,
		make([]string, 0),
	}, nil
}

func (s *Storage) SetSearchWeights(tag, visual, text float32) {
	s.tagSearchWeight = tag
	s.visualSearchWeight = visual
	s.textSearchWeight = text
}
