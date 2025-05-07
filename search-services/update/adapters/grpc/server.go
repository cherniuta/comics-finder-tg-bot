package grpc

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	updatepb "yadro.com/course/proto/update"
	"yadro.com/course/update/core"
)

func NewServer(service core.Updater) *Server {
	return &Server{service: service}
}

type Server struct {
	updatepb.UnimplementedUpdateServer
	service core.Updater
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *Server) Status(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatusReply, error) {
	serverStatus := s.service.Status(ctx)

	switch serverStatus {
	case core.StatusIdle:
		return &updatepb.StatusReply{Status: updatepb.Status_STATUS_IDLE}, nil
	case core.StatusRunning:
		return &updatepb.StatusReply{Status: updatepb.Status_STATUS_RUNNING}, nil
	}
	return nil, status.Error(codes.Internal, "unknown status from service")
}

func (s *Server) Update(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Update(ctx); err != nil {
		if errors.Is(err, core.ErrAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "update already runs")
		}
		return nil, err
	}
	return nil, nil

}

func (s *Server) Stats(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatsReply, error) {
	serverStats, err := s.service.Stats(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get service stats: %d", err)
	}
	return &updatepb.StatsReply{
		WordsTotal:    int64(serverStats.DBStats.WordsTotal),
		WordsUnique:   int64(serverStats.DBStats.WordsUnique),
		ComicsFetched: int64(serverStats.ComicsFetched),
		ComicsTotal:   int64(serverStats.ComicsTotal)}, nil
}

func (s *Server) Drop(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	err := s.service.Drop(ctx)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
