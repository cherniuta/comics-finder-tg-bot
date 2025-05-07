package main

import (
	"context"
	"flag"
	"log"
	"maps"
	"net"
	"slices"
	"strings"
	"unicode"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/kljensen/snowball/english"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	wordspb "yadro.com/course/proto/words"
)

const (
	maxPhraseLen = 4096
)

type server struct {
	wordspb.UnimplementedWordsServer
}

func (s *server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *server) Norm(_ context.Context, in *wordspb.WordsRequest) (*wordspb.WordsReply, error) {

	if len(in.GetPhrase()) > maxPhraseLen {
		return nil, status.Error(codes.ResourceExhausted, "message size over 4KiB")
	}

	if in.GetPhrase() == "" {
		return nil, status.Error(codes.ResourceExhausted, "message is empty")
	}
	words := strings.FieldsFunc(in.GetPhrase(), func(r rune) bool {
		return !unicode.IsDigit(r) && !unicode.IsLetter(r)
	})

	stemmedWords := make(map[string]bool)

	for _, word := range words {
		w := strings.ToLower(word)
		if !english.IsStopWord(w) {
			stemmed := english.Stem(w, false)
			if stemmed != "" {
				stemmedWords[stemmed] = true
			}
		}
	}

	return &wordspb.WordsReply{
		Words: slices.Collect(maps.Keys(stemmedWords)),
	}, nil
}

type Config struct {
	Address string `yaml:"words_address" env:"WORDS_ADDRESS" env-default:"80"`
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic(err)
	}

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	wordspb.RegisterWordsServer(s, &server{})
	reflection.Register(s)

	if err := s.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
