package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	photolib "github.com/daywee-zx/go-photo-library/pkg/photoLib"

	"github.com/daywee-zx/go-photo-library/pkg/aimanip"
	"github.com/daywee-zx/go-photo-library/pkg/storage"
)

func main() {
	dbPath := "test_images/test.db"

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return
	}
	storage := storage.NewStorage(db)
	if err != nil {
		fmt.Printf("Error initializing storage: %v\n", err)
		return
	}
	defer storage.Close()

	err = storage.Init()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		return
	}

	ctx := context.Background()

	tagWeight, visualWeight, textWeight := 0.2, 0.4, 0.4

	lib := photolib.NewPhotoLib(
		ctx,
		storage,
		aimanip.Embedder{
			ModelName: "bge-m3",
			URL:       "http://localhost:11434/api/embed",
		},
		aimanip.Tagger{
			ModelName: "qwen2.5vl",
			URL:       "http://localhost:11434/api/chat",
		},
		photolib.NewSearchConfig(float32(tagWeight), float32(visualWeight), float32(textWeight)),
	)

	path := "test_images/test%d.jpg"

	for i := 1; i <= 4; i++ {
		nowPath := fmt.Sprintf(path, i)
		id, err := lib.AddImage(nowPath)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("Image %s successfully inserted at id %d\n", nowPath, id)
	}

	testCases := []struct {
		request    string
		expectedID int64
	}{
		{"comic about soldiers", 1},
		{"beautiful landscape", 4},
		{"trolley problem", 3},
		{"я вам квартиру без черкашей сдавала", 2},
		{"а где граница тащ старшина", 1},
		{"no more psychological dylemma", 3},
		{"where is the border sergeant", 1},
	}

	for _, v := range testCases {
		now := time.Now()
		entries, err := lib.Search(v.request, 1)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Request: %s. Want: %d. Result: %d. Time elapsed: %v\n", v.request, v.expectedID, entries[0].ID, time.Since(now))
	}

	for i := 1; i <= 4; i++ {

		if err = lib.DeleteEntry(int64(i)); err != nil {
			fmt.Printf("Error occured while deleting: %v\n", err)
		}
		fmt.Printf("Successfully deleted id %d\n", i)
	}
}
