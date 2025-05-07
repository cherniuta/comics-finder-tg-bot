package core

import "context"

type APIClient interface {
	Search(ctx context.Context, limit int, words string) (SearchResult, error)
	Login(ctx context.Context, user, password string) (string, error)
	UpdateComics(ctx context.Context, token string) error
	Drop(ctx context.Context, token string) error
	Stats(ctx context.Context, token string) (StatsResult, error)
}

type TelegramClient interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
	GetUpdatesChan() <-chan TelegramUpdate
}
