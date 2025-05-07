package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"yadro.com/course/telegram/adapters/api"
	"yadro.com/course/telegram/adapters/rest"
	"yadro.com/course/telegram/adapters/telegram"
	"yadro.com/course/telegram/config"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)

	tgToken := cfg.TgToken
	if tgToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	log := mustMakeLogger(cfg.LogLevel)

	log.Info("starting server")
	log.Debug("debug messages are enabled")

	tgClient := telegram.NewBotClient(tgToken)
	apiClient := api.New("http://"+cfg.APIClient, log)

	handler := rest.New(apiClient, tgClient, log)

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	updates := tgClient.GetUpdatesChan()

	log.Info("Bot started, waiting for messages...")
	for {
		select {
		case <-ctx.Done():
			log.Info("Shutting down bot...")
			return
		case update := <-updates:
			if update.Message == nil {
				continue
			}

			if strings.HasPrefix(update.Message.Text, "/") {
				cmd, _ := parseCommand(update.Message.Text)
				if err := handler.HandleCommand(ctx, cmd, update.Message.Chat.ID); err != nil {
					log.Error("Error handling command", "error", err)
					_ = tgClient.SendMessage(
						context.Background(),
						update.Message.Chat.ID,
						"Произошла ошибка при обработке команды",
					)
				}
			} else {
				if err := handler.HandleRegularMessage(ctx, update.Message.Text, update.Message.Chat.ID); err != nil {
					log.Error("Error handling command", "error", err)
					_ = tgClient.SendMessage(
						context.Background(),
						update.Message.Chat.ID,
						"Произошла ошибка при обработке команды",
					)
				}
			}
		}
	}
}

func parseCommand(text string) (cmd, args string) {
	if !strings.HasPrefix(text, "/") {
		return "", text
	}

	parts := strings.SplitN(strings.TrimSpace(text), " ", 2)
	if len(parts) == 0 {
		return "", ""
	}

	cmd = parts[0]
	if len(parts) > 1 {
		args = parts[1]
	}
	return cmd, args
}

func mustMakeLogger(logLevel string) *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + logLevel)
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level, AddSource: true})
	return slog.New(handler)
}
