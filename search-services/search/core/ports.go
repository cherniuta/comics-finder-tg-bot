package core

import "context"

type Searcher interface {
	Search(ctx context.Context, query SearchQuery) ([]Comics, error)
	SearchIndex(ctx context.Context, query SearchQuery) ([]Comics, error)
	BuildIndex(ctx context.Context) error
}

type DB interface {
	CheckDB() error
	Search(ctx context.Context, keyword string) ([]int, error)
	Get(ctx context.Context, id int) (Comics, error)
	MaxId(ctx context.Context) (int, error)
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}
