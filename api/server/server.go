package server

import (
	"context"
	"os"
	"strconv"
	"time"

	photolib "github.com/daywee-zx/go-photo-library/pkg/photoLib"
	"github.com/daywee-zx/go-photo-library/pkg/storage"
	api "github.com/daywee-zx/go-photolib-grpc/api/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	timeFormat = "200601021504"
)

type Server struct {
	api.UnimplementedPhotoLibraryServer

	lib       *photolib.PhotoLib
	imagePath string
	timeout   time.Duration
}

func NewServer(lib *photolib.PhotoLib, imagePath string) *Server {
	return &Server{lib: lib, imagePath: imagePath}
}

func (s *Server) AddImage(ctx context.Context, req *api.AddImageRequest) (*api.SingleID, error) {
	now := time.Now()
	nowStr := now.Format(timeFormat)
	fileName := "img" + strconv.FormatInt(req.User, 10) + "_" + nowStr + ".jpg"
	filePath := s.imagePath + fileName

	// TODO: save as temp
	file, err := os.Create(filePath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "internal error while saving image")
	}
	defer file.Close()

	if _, err := file.Write(req.Image); err != nil {
		return nil, status.Errorf(codes.Internal, "internal error while saving image")
	}

	entry := storage.Entry{
		Path:      filePath,
		Tags:      req.Tags,
		CreatedAt: now,
		UserID:    req.User,
	}

	subCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	id, err := s.lib.AddImage(subCtx, entry)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "internal error while saving entry")
	}

	return &api.SingleID{Id: id}, nil
}

func (s *Server) GetImage(ctx context.Context, req *api.SingleID) (*api.EntryWithImage, error) {
	entry, err := s.lib.GetEntry(ctx, req.Id)
	// TODO: add not found
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error while getting image path")
	}

	data, err := os.ReadFile(entry.Path)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error while reading the image")
	}

	resp := &api.EntryWithImage{
		Entry: entryStorageToProto(entry),
		Image: data,
	}

	return resp, nil
}

func (s *Server) DeleteEntry(ctx context.Context, req *api.SingleID) (*emptypb.Empty, error) {
	err := s.lib.DeleteEntry(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error while deleting entry")
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) GetEntry(ctx context.Context, req *api.SingleID) (*api.Entry, error) {
	entry, err := s.lib.GetEntry(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error while getting entry")
	}
	return entryStorageToProto(entry), nil
}

func (s *Server) GetEntryTags(ctx context.Context, req *api.SingleID) (*api.TagsResponse, error) {
	tags, err := s.lib.GetEntryTags(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error while getting entry tags")
	}
	return &api.TagsResponse{Tags: tags}, nil
}

func (s *Server) GetAvailableTags(ctx context.Context, req *emptypb.Empty) (*api.TagsResponse, error) {
	tags, err := s.lib.GetAvailableTags(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error while getting all tags")
	}
	return &api.TagsResponse{Tags: tags}, nil
}

func (s *Server) Search(ctx context.Context, req *api.SearchRequest) (*api.SearchResponse, error) {
	result, err := s.lib.Search(ctx, req.Query, int(req.TopK))
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error while searching")
	}
	protoRes := make([]*api.Entry, 0)
	for _, r := range result {
		protoRes = append(protoRes, entryStorageToProto(r))
	}
	return &api.SearchResponse{Result: protoRes}, nil
}

func entryStorageToProto(entry storage.Entry) *api.Entry {
	return &api.Entry{
		Id:        entry.ID,
		Tags:      entry.Tags,
		CreatedAt: timestamppb.New(entry.CreatedAt),
		User:      entry.UserID,
		Desc:      entry.Description,
	}
}
