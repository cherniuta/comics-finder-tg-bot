package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	seachpb "yadro.com/course/proto/search"
	"yadro.com/course/search/core"
)

func NewServer(service core.Searcher) *Server {
	return &Server{service: service}
}

type Server struct {
	seachpb.UnimplementedSearchServer
	service core.Searcher
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *Server) Search(ctx context.Context, in *seachpb.SearchRequest) (*seachpb.SearchReply, error) {
	searchQuery := core.SearchQuery{Keywords: in.Keywords, Limit: int(in.Limit)}
	replay, err := s.service.Search(ctx, searchQuery)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "nothing found")
		}
	}

	comics := make([]*seachpb.Comics, 0)
	for _, index := range replay {
		comics = append(comics, &seachpb.Comics{Id: int64(index.ID), Url: index.URL})
	}

	return &seachpb.SearchReply{Comics: comics}, err

}

func (s *Server) SearchIndex(ctx context.Context, in *seachpb.SearchRequest) (*seachpb.SearchReply, error) {
	searchQuery := core.SearchQuery{Keywords: in.Keywords, Limit: int(in.Limit)}
	replay, err := s.service.SearchIndex(ctx, searchQuery)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "nothing found")
		}
	}

	comics := make([]*seachpb.Comics, 0)
	for _, index := range replay {
		comics = append(comics, &seachpb.Comics{Id: int64(index.ID), Url: index.URL})
	}

	return &seachpb.SearchReply{Comics: comics}, err

}
