package main

import (
	"fmt"
	"memeLibrary/pkg/aimanip"
)

func main() {
	/*
		path := "test_images/test%d.jpg"

		for i := 1; i <= 1; i++ {
			imagePath := fmt.Sprintf(path, i)
			description, err := aimanip.TagImage(imagePath)
			if err != nil {
				fmt.Printf("Error tagging image %s: %v\n", imagePath, err)
				continue
			}
			fmt.Printf("Image: %s\nDescription: %s\n\n", imagePath, description)
		}*/

	text1 := "The quick brown fox jumps over the lazy dog."
	text2 := "Быстрая коричневая лиса перепрыгивает через ленивую собаку."

	embeddings, err := aimanip.Embed([]string{text1, text2})
	if err != nil {
		fmt.Printf("Error embedding text: %v\n", err)
		return
	}

	similarity := aimanip.CosineSimilarity(embeddings[0], embeddings[1])
	fmt.Printf("Cosine similarity between the two texts: %f\n", similarity)
	fmt.Printf("Length of embedding 1: %d\n", len(embeddings[0]))
	fmt.Printf("Length of embedding 2: %d\n", len(embeddings[1]))
}
