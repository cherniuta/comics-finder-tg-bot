package rest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"yadro.com/course/telegram/core"
)

type Handler struct {
	apiClient   core.APIClient
	tgClint     core.TelegramClient
	userStates  map[int64]*core.UserState
	adminTokens map[int64]string
	stateMu     sync.RWMutex
	tokenMu     sync.RWMutex
	log         *slog.Logger
}

func New(apiClient core.APIClient, tgClient core.TelegramClient, logger *slog.Logger) *Handler {
	return &Handler{apiClient: apiClient, tgClint: tgClient, userStates: make(map[int64]*core.UserState), adminTokens: make(map[int64]string), log: logger}
}

func (h *Handler) HandleCommand(ctx context.Context, cmd string, chatID int64) error {
	switch cmd {
	case "/search":
		h.stateMu.Lock()
		h.userStates[chatID] = &core.UserState{Step: "limit"}
		h.stateMu.Unlock()

		return h.tgClint.SendMessage(ctx, chatID, "Введите сколько комикосв вы бы хотели найти")
	case "/help":
		return h.sendHelp(ctx, chatID)
	case "/start":
		return h.sendHello(ctx, chatID)
	case "/admin":
		h.stateMu.Lock()
		h.userStates[chatID] = &core.UserState{Step: "login"}
		h.stateMu.Unlock()

		return h.tgClint.SendMessage(ctx, chatID, "Введите ваш логин")
	case "/update":
		token, err := h.GetAdminToken(chatID)
		if err != nil {
			return h.tgClint.SendMessage(ctx, chatID, "У вас нет права доступа к данной операции")
		}

		if err := h.apiClient.UpdateComics(ctx, token); err != nil {
			if errors.Is(err, core.ErrAlreadyExists) {
				return h.tgClint.SendMessage(ctx, chatID, "Обновление уже выполняется")
			} else if errors.Is(err, core.ErrUnauthorized) {
				return h.tgClint.SendMessage(ctx, chatID, "Время вашего токена истекло")
			}
			return h.tgClint.SendMessage(ctx, chatID, "Ошибка обновления ")
		}

		return h.tgClint.SendMessage(ctx, chatID, "База данных успешно обновлена")
	case "/drop":
		token, err := h.GetAdminToken(chatID)
		if err != nil {
			return h.tgClint.SendMessage(ctx, chatID, "У вас нет права доступа к данной операции")
		}

		if err := h.apiClient.Drop(ctx, token); err != nil {
			if errors.Is(err, core.ErrUnauthorized) {
				return h.tgClint.SendMessage(ctx, chatID, "Время вашего токена истекло")
			}
			return h.tgClint.SendMessage(ctx, chatID, "Ошибка удаления")
		}

		return h.tgClint.SendMessage(ctx, chatID, "База данных успешно очищена")
	case "/stats":
		token, err := h.GetAdminToken(chatID)
		if err != nil {
			return h.tgClint.SendMessage(ctx, chatID, "У вас нет права доступа к данной операции")
		}
		result, err := h.apiClient.Stats(ctx, token)
		if err != nil {
			if errors.Is(err, core.ErrUnauthorized) {
				return h.tgClint.SendMessage(ctx, chatID, "Время вашего токена истекло")
			}
			return h.tgClint.SendMessage(ctx, chatID, "Ошибка получения информации от сервера")
		}
		return h.sendStatsResult(ctx, chatID, result)
	default:
		return h.sendUnknownCommand(ctx, chatID)
	}
}

func (h *Handler) HandleRegularMessage(ctx context.Context, text string, chatID int64) error {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()

	state, exists := h.userStates[chatID]
	if !exists {
		return h.sendUnknownCommand(ctx, chatID)
	}

	switch state.Step {
	case "limit":
		var (
			limit int
			err   error
		)
		limit, err = strconv.Atoi(text)
		if err != nil || limit <= 0 {
			return h.tgClint.SendMessage(ctx, chatID, "Введите число больше 0")
		}
		state.Limit = limit
		state.Step = "phrase"
		return h.tgClint.SendMessage(ctx, chatID, "Введите на какую тему вы бы хотели найти комиксы")
	case "phrase":
		state.Phrase = text
		defer delete(h.userStates, chatID)

		results, err := h.apiClient.Search(ctx, state.Limit, state.Phrase)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}
		return h.sendComicsResults(ctx, chatID, results)
	case "login":
		state.User = text
		state.Step = "password"

		return h.tgClint.SendMessage(ctx, chatID, "Введите пароль")
	case "password":
		state.Password = text

		defer delete(h.userStates, chatID)

		results, err := h.apiClient.Login(ctx, state.User, state.Password)
		if err != nil {
			return h.tgClint.SendMessage(ctx, chatID, "Неверные данные")
		}

		h.log.Info("sendLoginResults")
		return h.sendLoginResults(ctx, chatID, results)

	default:
		return h.sendUnknownCommand(ctx, chatID)

	}
}

func (h *Handler) sendHelp(ctx context.Context, chatID int64) error {
	return h.tgClint.SendMessage(ctx, chatID, msgHelp)
}

func (h *Handler) sendHello(ctx context.Context, chatID int64) error {
	return h.tgClint.SendMessage(ctx, chatID, msgHello)
}

func (h *Handler) sendUnknownCommand(ctx context.Context, chatID int64) error {
	return h.tgClint.SendMessage(ctx, chatID, msgUnknownCommand)
}
func (h *Handler) sendStatsResult(ctx context.Context, chatId int64, result core.StatsResult) error {
	var builder strings.Builder

	builder.WriteString("Информация о комиксах:\n\n")

	builder.WriteString(fmt.Sprintf("Уникальных слов:%d\n", result.WordsUnique))
	builder.WriteString(fmt.Sprintf("Скачено комиксов:%d\n", result.ComicsFetched))
	builder.WriteString(fmt.Sprintf("Всего комиксов:%d\n", result.ComicsTotal))

	msg := builder.String()

	return h.tgClint.SendMessage(ctx, chatId, msg)
}
func formatResultsHTML(results core.SearchResult) string {
	var builder strings.Builder
	if results.Total == 0 {
		return "Ничего не найдено"
	}

	builder.WriteString("Результаты поиска:\n\n")

	for i, item := range results.Comics {
		builder.WriteString(fmt.Sprintf("%d. %s #%d\n",
			i+1, item.URL, item.ID))
	}
	return builder.String()
}
func (h *Handler) sendLoginResults(ctx context.Context, chatId int64, token string) error {
	h.log.Info("SetAdminToken")
	h.SetAdminToken(chatId, token)
	return h.tgClint.SendMessage(ctx, chatId, "У вас теперь есть права админа")
}
func (h *Handler) sendComicsResults(ctx context.Context, chatId int64, result core.SearchResult) error {
	msg := formatResultsHTML(result)
	return h.tgClint.SendMessage(ctx, chatId, msg)
}

func (h *Handler) SetAdminToken(chatID int64, token string) {
	h.tokenMu.Lock()
	defer h.tokenMu.Unlock()

	h.adminTokens[chatID] = token
}

func (h *Handler) GetAdminToken(chatID int64) (string, error) {
	h.tokenMu.RLock()
	defer h.tokenMu.RUnlock()
	token, exists := h.adminTokens[chatID]
	if !exists {
		return "", errors.New("admin token not found for this user")
	}
	return token, nil
}
