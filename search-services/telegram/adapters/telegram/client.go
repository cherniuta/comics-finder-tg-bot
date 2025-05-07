package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"yadro.com/course/telegram/core"
)

type BotClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewBotClient(token string) *BotClient {
	return &BotClient{
		baseURL:    "https://api.telegram.org",
		token:      token,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (b *BotClient) SendMessage(ctx context.Context, chatID int64, text string) error {
	params := url.Values{}
	params.Add("chat_id", strconv.FormatInt(chatID, 10))
	params.Add("text", text)

	_, err := b.doRequest(ctx, "sendMessage", params)
	return err
}

func (b *BotClient) GetUpdatesChan() <-chan core.TelegramUpdate {
	updates := make(chan core.TelegramUpdate, 100)

	go func() {
		offset := 0
		for {
			updatesBatch, err := b.getUpdates(context.Background(), offset, 100)
			if err != nil {
				fmt.Printf("Error getting updates: %v\n", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for _, update := range updatesBatch {
				updates <- update
				offset = int(update.UpdateID) + 1
			}

			time.Sleep(1 * time.Second)
		}
	}()

	return updates
}

func (b *BotClient) getUpdates(ctx context.Context, offset, limit int) ([]core.TelegramUpdate, error) {
	params := url.Values{}
	params.Add("offset", strconv.Itoa(offset))
	params.Add("limit", strconv.Itoa(limit))

	data, err := b.doRequest(ctx, "getUpdates", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		OK     bool                  `json:"ok"`
		Result []core.TelegramUpdate `json:"result"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("telegram API error")
	}

	return response.Result, nil
}

func (b *BotClient) doRequest(ctx context.Context, method string, params url.Values) ([]byte, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.telegram.org",
		Path:   path.Join("bot"+b.token, method),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.URL.RawQuery = params.Encode()
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	return data, nil
}
