package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	ID          int64     `json:"id"`
	Path        string    `json:"path"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UserID      int64     `json:"user"`
	Description string    `json:"description"`
}

type IndexedEntry struct {
	Entry

	VisualEmbed []float32 `json:"visual_embedding"`
	TextEmbed   []float32 `json:"ocr_embedding"`
}

func NewStorage(db *sql.DB) *Storage {
	return &Storage{
		db:                 db,
		tagSearchWeight:    defaultTagWeight,
		visualSearchWeight: defaultVisualWeight,
		textSearchWeight:   defaultTextWeight,
	}
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) Init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL UNIQUE,
			user_id INTEGER,
			created_at TIMESTAMP,
			desc TEXT
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
			FOREIGN KEY (entry_id) REFERENCES entries(id) ON DELETE CASCADE,
			FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_entry_tags_tag ON entry_tags(tag_id);`)
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

	_, err = s.db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) InsertEntry(e IndexedEntry) (int64, error) {
	if e.Path == "" {
		return 0, fmt.Errorf("no path for entry specified")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO entries (path, created_at, user_id, desc) VALUES (?,?,?,?)
	`, e.Path, e.CreatedAt, e.UserID, e.Description)
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

func (s *Storage) GetEntry(ctx context.Context, entryID int64) (Entry, error) {
	if entryID < 0 {
		return Entry{}, fmt.Errorf("id can not be negative")
	}

	query := `
		SELECT e.path, e.created_at, e.desc, e.user_id, GROUP_CONCAT(t.name, ', ') AS tags
		FROM entries e
		LEFT JOIN entry_tags et ON et.entry_id = e.id
		LEFT JOIN tags t ON t.id = et.tag_id
		WHERE e.id = ?
		GROUP BY e.id, e.path
	`

	row := s.db.QueryRowContext(ctx, query, entryID)

	var path, desc string
	var createdAt time.Time
	var userID int64
	var tags sql.NullString

	err := row.Scan(&path, &createdAt, &desc, &userID, &tags)
	if err != nil {
		return Entry{}, err
	}

	var tagList []string
	if tags.Valid {
		tagList = strings.Split(tags.String, ", ")
	}

	return Entry{
		entryID,
		path,
		tagList,
		createdAt,
		userID,
		desc,
	}, nil
}

func (s *Storage) DeleteEntry(entryID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM visual_embeddings WHERE rowid = ?`, entryID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM ocr_embeddings WHERE rowid = ?`, entryID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM entries WHERE id = ?`, entryID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		DELETE FROM tags
		WHERE id NOT IN (
			SELECT DISTINCT tag_id
			FROM entry_tags
		)
	`)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Storage) GetEntryTags(ctx context.Context, entryID int64) ([]string, error) {
	if entryID < 0 {
		return nil, fmt.Errorf("id can not be negative")
	}

	query := `
		SELECT GROUP_CONCAT(t.name, ', ') AS tags
		FROM entry_tags et
		LEFT JOIN tags t ON t.id = et.tag_id
		WHERE et.entry_id = ?
	`
	row := s.db.QueryRowContext(ctx, query, entryID)
	if row.Err() != nil {
		return nil, row.Err()
	}

	var tags sql.NullString

	err := row.Scan(&tags)
	if err != nil {
		return nil, err
	}

	var tagList []string
	if tags.Valid {
		tagList = strings.Split(tags.String, ", ")
	}

	return tagList, nil
}

func (s *Storage) GetAvailableTags(ctx context.Context) ([]string, error) {
	query := `
		SELECT GROUP_CONCAT(name, ', ') AS tags
		FROM tags
	`

	row := s.db.QueryRowContext(ctx, query)

	if row.Err() != nil {
		return nil, row.Err()
	}

	var tags sql.NullString

	err := row.Scan(&tags)
	if err != nil {
		return nil, err
	}

	var tagList []string
	if tags.Valid {
		tagList = strings.Split(tags.String, ", ")
	}

	return tagList, nil
}

func (s *Storage) SetSearchWeights(tag, visual, text float32) {
	s.tagSearchWeight = tag
	s.visualSearchWeight = visual
	s.textSearchWeight = text
}
