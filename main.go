package main

import (
	"fmt"
	"memeLibrary/pkg/aimanip"
	"memeLibrary/pkg/storage"
	"time"
)

func main() {

	/*path := "test_images/test%d.jpg"
	entries := []storage.IndexedEntry{}

	for i := 1; i <= 4; i++ {
		imagePath := fmt.Sprintf(path, i)
		tagResp, err := aimanip.TagImage(imagePath)
		if err != nil {
			fmt.Printf("Error tagging image %s: %v\n", imagePath, err)
			continue
		}

		embeds, err := aimanip.Embed([]string{tagResp.Description, tagResp.Text})
		if err != nil {
			fmt.Printf("Error embedding image %s: %v\n", imagePath, err)
			continue
		}
		visual, ocd := embeds[0], embeds[1]
		fmt.Printf("visual embedding length: %d, ocd embedding length: %d\n", len(visual), len(ocd))
		fmt.Printf("Image: %s\nDescription: %s\n\n", imagePath, tagResp.Description)

		entries = append(entries, storage.IndexedEntry{
			Entry: storage.Entry{
				ID:   int64(i),
				Path: imagePath,
				Tags: tagResp.Tags,
			},
			VisualEmbed: visual,
			OCDEmbed:    ocd,
		})
	}*/

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

	/*
		for _, entry := range entries {
			err = storage.InsertEntry(entry)
			if err != nil {
				fmt.Printf("Error inserting entry for image %s: %v\n", entry.Path, err)
				continue
			}
			fmt.Printf("Inserted entry for image %s successfully.\n", entry.Path)
		}*/

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

		embeddings, err := aimanip.Embed([]string{v.request})
		if err != nil {
			fmt.Println(err)
		}
		embedding := embeddings[0]

		tags, err := aimanip.TagRequest(v.request)
		if err != nil {
			fmt.Println(err)
		}

		//fmt.Printf("Request: %s. Tags: %v\n", v.request, tags)

		now := time.Now()
		entry, err := storage.Search(tags, embedding)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Request: %s. Want: %d. Result: %d. Time elapsed: %v\n", v.request, v.expectedID, entry.ID, time.Since(now))
	}
}
