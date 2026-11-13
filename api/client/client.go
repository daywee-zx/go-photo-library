package main

import (
	"context"
	"fmt"
	"log"
	"os"

	api "github.com/daywee-zx/go-photolib-grpc/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := api.NewPhotoLibraryClient(conn)
	ctx := context.Background()

	data, err := os.ReadFile("test1.jpg")
	if err != nil {
		log.Fatal(err)
	}

	resp1, err := client.AddImage(ctx, &api.AddImageRequest{
		Image: data,
		User:  1,
	})
	if err != nil {
		log.Fatal(err)
	}
	id := resp1.Id

	resp2, err := client.GetImage(ctx, &api.SingleID{Id: id})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp2)
}
