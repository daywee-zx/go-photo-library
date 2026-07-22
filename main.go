package main

import (
	"context"
	"fmt"
	"photoLibrary/pkg/aimanip"
	photolib "photoLibrary/pkg/photoLib"
	"photoLibrary/pkg/storage"
	"time"
)

func main() {
	dbPath := "test_images/test.db"
	storage, err := storage.NewStorage(dbPath)
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
		entry, err := lib.Search(v.request)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Request: %s. Want: %d. Result: %d. Time elapsed: %v\n", v.request, v.expectedID, entry.ID, time.Since(now))
	}
}
