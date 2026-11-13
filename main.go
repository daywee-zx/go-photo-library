package main

import (
	"context"
	"database/sql"
	"log"
	"net"

	api "github.com/daywee-zx/go-photo-library/api/proto"
	"github.com/daywee-zx/go-photo-library/api/server"
	"github.com/daywee-zx/go-photo-library/pkg/aimanip"
	photolib "github.com/daywee-zx/go-photo-library/pkg/photoLib"
	"github.com/daywee-zx/go-photo-library/pkg/storage"
	"google.golang.org/grpc"
)

func main() {
	// TODO: switch to .env
	imagePath := "data/images/"
	dbPath := "data/photo.db"
	embedderModel := "bge-m3"
	embedderURL := "http://localhost:11434/api/embed"
	taggerModel := "qwen2.5vl"
	taggerURL := "http://localhost:11434/api/chat"

	var tagWeight, visualWeight, textWeight float32
	tagWeight, visualWeight, textWeight = 0.2, 0.4, 0.4

	// start the storage
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	storage := storage.NewStorage(db)
	defer storage.Close()

	err = storage.Init()
	if err != nil {
		log.Fatal(err)
	}

	// configure ai APIs
	embedder := aimanip.Embedder{ModelName: embedderModel, URL: embedderURL}
	tagger := aimanip.Tagger{ModelName: taggerModel, URL: taggerURL}

	// create photolib
	ctx := context.Background()
	searchConfig := photolib.NewSearchConfig(tagWeight, visualWeight, textWeight)

	lib := photolib.NewPhotoLib(
		ctx,
		storage,
		embedder,
		tagger,
		searchConfig,
	)

	// start the server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()

	svc := server.NewServer(lib, imagePath)

	api.RegisterPhotoLibraryServer(grpcServer, svc)

	log.Println("gRPC server listening on :50051")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
